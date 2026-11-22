#include "MainWindow.h"

#include <QCheckBox>
#include <QClipboard>
#include <QCloseEvent>
#include <QComboBox>
#include <QDesktopServices>
#include <QEvent>
#include <QFileDialog>
#include <QFileInfo>
#include <QGuiApplication>
#include <QHBoxLayout>
#include <QInputDialog>
#include <QItemSelectionModel>
#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QLabel>
#include <QLineEdit>
#include <QMessageBox>
#include <QProgressBar>
#include <QPushButton>
#include <QSaveFile>
#include <QSettings>
#include <QSplitter>
#include <QStringList>
#include <QStatusBar>
#include <QStackedWidget>
#include <QTimer>
#include <QUrl>
#include <QVBoxLayout>

#include "../engine/EngineClient.h"
#include "../engine/EngineLocator.h"
#include "../model/ModTableModel.h"
#include "../model/ViewController.h"
#include "ChipBar.h"
#include "DetailsPane.h"
#include "ModTableView.h"
#include "PaletteUtil.h"
#include "ReportDialog.h"
#include "StartScreen.h"

namespace {

quint64 connectCall(EngineClient *engine, const QString &method, const QJsonObject &params,
                    QObject *context, std::function<void(quint64, bool, const QJsonObject &, const QJsonObject &)> fn)
{
    const quint64 id = engine->call(method, params);
    QObject::connect(engine, &EngineClient::response, context,
                     [id, fn = std::move(fn)](quint64 rid, bool ok, const QJsonObject &res, const QJsonObject &err) {
                         if (rid == id)
                             fn(rid, ok, res, err);
                     });
    return id;
}

const char *kSortFields[] = {"name", "id", "version", "author", "jar", "loader", "side", "status"};

} // namespace

MainWindow::MainWindow(QWidget *parent)
    : QMainWindow(parent)
    , m_engine(new EngineClient(this))
    , m_viewCtl(new ViewController(m_engine, this))
    , m_start(new StartScreen(this))
    , m_stack(new QStackedWidget(this))
    , m_status(new QLabel(this))
    , m_banner(new QWidget(this))
    , m_bannerText(new QLabel(m_banner))
    , m_bannerRestart(new QPushButton(tr("Restart"), m_banner))
    , m_bannerDetails(new QPushButton(tr("Details…"), m_banner))
{
    m_bannerDetails->setEnabled(false);
    setWindowTitle(tr("mcmodlist"));
    resize(1000, 640);

    m_stack->addWidget(m_start);
    buildMainView();
    m_stack->addWidget(m_mainView);
    m_stack->setCurrentWidget(m_start);

    auto *central = new QWidget(this);
    auto *layout = new QVBoxLayout(central);
    layout->setContentsMargins(8, 8, 8, 8);

    auto *bannerLayout = new QHBoxLayout(m_banner);
    bannerLayout->setContentsMargins(8, 4, 8, 4);
    bannerLayout->addWidget(m_bannerText, 1);
    bannerLayout->addWidget(m_bannerDetails);
    bannerLayout->addWidget(m_bannerRestart);
    m_banner->setVisible(false);
    layout->addWidget(m_banner);
    layout->addWidget(m_stack, 1);
    layout->addWidget(m_status);
    setCentralWidget(central);

    connect(m_start, &StartScreen::scanRequested, this, &MainWindow::onScanRequested);
    connect(m_bannerRestart, &QPushButton::clicked, this, &MainWindow::onRestart);
    connect(m_bannerDetails, &QPushButton::clicked, this, &MainWindow::onBannerDetails);
    connect(m_engine, &EngineClient::stderrLine, this, &MainWindow::onStderrLine);

    connect(m_engine, &EngineClient::engineError, this, &MainWindow::onEngineError);
    connect(m_engine, &EngineClient::engineExited, this, &MainWindow::onEngineExited);
    connect(m_engine, &EngineClient::protocolMismatch, this, &MainWindow::onProtocolMismatch);
    connect(m_engine, &EngineClient::eventReceived, this,
            [this](quint64 id, const QString &event, const QJsonObject &data) {
                if (id == m_scanRequestId && event == QLatin1String("scan-progress"))
                    onScanProgress(id, event, data);
            });
    connect(m_viewCtl, &ViewController::stateReady, this, &MainWindow::onStateReady);
    connect(m_viewCtl, &ViewController::viewError, this, &MainWindow::onViewError);

    m_searchDebounce = new QTimer(this);
    m_searchDebounce->setSingleShot(true);
    m_searchDebounce->setInterval(120);
    connect(m_searchDebounce, &QTimer::timeout, this, &MainWindow::applySearchNow);

    restoreSettings();
    statusBar()->showMessage(tr("Starting engine…"));
    startEngine();
}

void MainWindow::buildMainView()
{
    m_mainView = new QWidget(this);
    auto *layout = new QVBoxLayout(m_mainView);
    layout->setContentsMargins(0, 0, 0, 0);

    m_headerLabel = new QLabel(m_mainView);
    m_headerLabel->setWordWrap(true);
    layout->addWidget(m_headerLabel);

    auto *toolbar = new QWidget(m_mainView);
    auto *toolLayout = new QHBoxLayout(toolbar);
    toolLayout->setContentsMargins(0, 0, 0, 0);

    m_search = new QLineEdit(toolbar);
    m_search->setPlaceholderText(tr("Search…"));
    m_search->setClearButtonEnabled(true);
    m_search->setAccessibleName(tr("Search mods"));
    m_search->setAccessibleDescription(tr("Filter the mod list by text"));
    toolLayout->addWidget(m_search);
    connect(m_search, &QLineEdit::textChanged, this, &MainWindow::onSearchEdited);

    m_sideCombo = new QComboBox(toolbar);
    m_sideCombo->setAccessibleName(tr("Side filter"));
    m_sideCombo->addItem(tr("All sides"), QString());
    m_sideCombo->addItem(QStringLiteral("client"), QStringLiteral("client"));
    m_sideCombo->addItem(QStringLiteral("server"), QStringLiteral("server"));
    m_sideCombo->addItem(QStringLiteral("both"), QStringLiteral("both"));
    toolLayout->addWidget(m_sideCombo);
    connect(m_sideCombo, &QComboBox::currentIndexChanged, this, [this]() { refreshView(); });

    m_chipBar = new ChipBar(toolbar);
    toolLayout->addWidget(m_chipBar, 1);
    connect(m_chipBar, &ChipBar::selectionChanged, this, [this](const QString &label) {
        auto opts = m_viewCtl->options();
        opts.loader = label;
        m_viewCtl->setOptions(opts);
    });

    m_disabledCheck = new QCheckBox(tr("Disabled"), toolbar);
    m_nestedCheck = new QCheckBox(tr("Nested"), toolbar);
    m_unrecCheck = new QCheckBox(tr("Unrecognised"), toolbar);
    m_disabledCheck->setAccessibleName(tr("Show disabled mods"));
    m_nestedCheck->setAccessibleName(tr("Show nested mods"));
    m_unrecCheck->setAccessibleName(tr("Show only unrecognised entries"));
    for (QCheckBox *check : {m_disabledCheck, m_nestedCheck, m_unrecCheck}) {
        toolLayout->addWidget(check);
        connect(check, &QCheckBox::toggled, this, [this]() { refreshView(); });
    }

    m_groupCombo = new QComboBox(toolbar);
    m_groupCombo->setAccessibleName(tr("Grouping"));
    m_groupCombo->addItem(tr("No grouping"), QString());
    m_groupCombo->addItem(tr("Loader"), QStringLiteral("loader"));
    m_groupCombo->addItem(tr("Side"), QStringLiteral("side"));
    toolLayout->addWidget(m_groupCombo);
    connect(m_groupCombo, &QComboBox::currentIndexChanged, this, [this]() { refreshView(); });

    auto *actions = new QHBoxLayout;
    m_checkBtn = new QPushButton(tr("Check"), toolbar);
    m_compareBtn = new QPushButton(tr("Compare…"), toolbar);
    m_copyBtn = new QPushButton(tr("Copy Markdown"), toolbar);
    m_exportBtn = new QPushButton(tr("Export…"), toolbar);
    m_checkBtn->setAccessibleName(tr("Check for problems"));
    m_compareBtn->setAccessibleName(tr("Compare with another instance"));
    m_copyBtn->setAccessibleName(tr("Copy mod list as Markdown"));
    m_exportBtn->setAccessibleName(tr("Export mod list to a file"));
    for (QPushButton *btn : {m_checkBtn, m_compareBtn, m_copyBtn, m_exportBtn})
        actions->addWidget(btn);
    actions->addStretch(1);
    toolLayout->addLayout(actions);

    layout->addWidget(toolbar);

    m_splitter = new QSplitter(Qt::Horizontal, m_mainView);
    m_model = new ModTableModel(m_mainView);
    m_table = new ModTableView(m_splitter);
    m_table->setModel(m_model);
    m_table->setAlternatingRowColors(true);
    m_table->setAccessibleName(tr("Mod list"));
    connect(m_table, &ModTableView::sortRequested, this, &MainWindow::onSortRequested);
    connect(m_table->selectionModel(), &QItemSelectionModel::currentRowChanged,
            this, &MainWindow::onSelectionChanged);
    m_splitter->addWidget(m_table);

    m_details = new DetailsPane(m_splitter);
    m_details->setMinimumWidth(220);
    m_splitter->addWidget(m_details);
    m_splitter->setStretchFactor(0, 3);
    m_splitter->setStretchFactor(1, 1);
    layout->addWidget(m_splitter, 1);

    auto *progressRow = new QHBoxLayout;
    m_progress = new QProgressBar(m_mainView);
    m_progress->setRange(0, 0);
    m_progress->setVisible(false);
    m_cancelBtn = new QPushButton(tr("Cancel"), m_mainView);
    m_cancelBtn->setAccessibleName(tr("Cancel scan"));
    m_cancelBtn->setVisible(false);
    progressRow->addWidget(m_progress, 1);
    progressRow->addWidget(m_cancelBtn);
    connect(m_cancelBtn, &QPushButton::clicked, this, &MainWindow::onCancelScan);
    layout->addLayout(progressRow);

    connect(m_checkBtn, &QPushButton::clicked, this, &MainWindow::onCheck);
    connect(m_compareBtn, &QPushButton::clicked, this, &MainWindow::onCompare);
    connect(m_copyBtn, &QPushButton::clicked, this, &MainWindow::onCopyMarkdown);
    connect(m_exportBtn, &QPushButton::clicked, this, &MainWindow::onExport);
}

bool MainWindow::event(QEvent *event)
{
    if (event->type() == QEvent::ApplicationPaletteChange || event->type() == QEvent::PaletteChange) {
        PaletteUtil::invalidate();
        m_start->update();
        if (m_mainView)
            m_mainView->update();
        update();
    }
    return QMainWindow::event(event);
}

void MainWindow::closeEvent(QCloseEvent *event)
{
    saveSettings();
    QMainWindow::closeEvent(event);
}

void MainWindow::restoreSettings()
{
    QSettings settings;
    restoreGeometry(settings.value(QStringLiteral("geometry")).toByteArray());
    const QByteArray splitterState = settings.value(QStringLiteral("splitter")).toByteArray();
    if (!splitterState.isEmpty() && m_splitter)
        m_splitter->restoreState(splitterState);
    m_lastPath = settings.value(QStringLiteral("lastPath")).toString();
    if (!m_lastPath.isEmpty())
        m_start->setPath(m_lastPath);
}

void MainWindow::saveSettings() const
{
    QSettings settings;
    settings.setValue(QStringLiteral("geometry"), saveGeometry());
    if (m_splitter)
        settings.setValue(QStringLiteral("splitter"), m_splitter->saveState());
    if (!m_lastPath.isEmpty())
        settings.setValue(QStringLiteral("lastPath"), m_lastPath);
}

void MainWindow::onBannerDetails()
{
    if (m_stderrLines.isEmpty()) {
        m_bannerDetails->setEnabled(false);
        return;
    }
    ReportDialog dlg(tr("Engine log"), m_stderrLines, this);
    dlg.exec();
}

void MainWindow::onStderrLine(const QString &line)
{
    if (m_stderrLines.size() >= 200)
        m_stderrLines.removeFirst();
    m_stderrLines.append(line);
    m_bannerDetails->setEnabled(true);
}

void MainWindow::startEngine()
{
    clearBanner();
    const QString path = findEnginePath();
    if (path.isEmpty()) {
        showBanner(tr("mcmodlist-engine not found. Set MCMODLIST_ENGINE or install the engine next to this app."));
        setStatus(tr("Engine not found"));
        return;
    }
    m_engine->setEnginePath(path);
    // Persist an explicit env override so it survives restarts.
    if (!qEnvironmentVariable("MCMODLIST_ENGINE").isEmpty())
        QSettings().setValue(QStringLiteral("enginePath"), path);
    if (!m_engine->start()) {
        showBanner(tr("Could not start engine: %1").arg(path));
        return;
    }
    setStatus(tr("Engine started: %1").arg(path));
    m_helloId = m_engine->callHello();
    QObject::connect(m_engine, &EngineClient::response, this,
                     [this](quint64 id, bool ok, const QJsonObject &res, const QJsonObject &err) {
                         if (id == m_helloId)
                             onHello(id, ok, res, err);
                     });
}

void MainWindow::onHello(quint64, bool ok, const QJsonObject &, const QJsonObject &error)
{
    if (!ok) {
        showBanner(tr("Engine handshake failed: %1").arg(error.value(QStringLiteral("message")).toString()));
        return;
    }
    connectCall(m_engine, QStringLiteral("listInstances"), {}, this,
                [this](quint64, bool ok2, const QJsonObject &res, const QJsonObject &err2) {
                    onListInstances(0, ok2, res, err2);
                });
}

void MainWindow::onListInstances(quint64, bool ok, const QJsonObject &result, const QJsonObject &error)
{
    if (!ok) {
        setStatus(tr("Could not list instances: %1").arg(error.value(QStringLiteral("message")).toString()));
        return;
    }
    QList<QPair<QString, QString>> items;
    const QJsonArray arr = result.value(QStringLiteral("instances")).toArray();
    for (const QJsonValue &v : arr) {
        const QJsonObject o = v.toObject();
        const QString name = o.value(QStringLiteral("name")).toString();
        const QString path = o.value(QStringLiteral("path")).toString();
        if (!path.isEmpty())
            items.append({name.isEmpty() ? path : name, path});
    }
    m_start->setInstances(items);
    setStatus(tr("%1 instance(s) found").arg(items.size()));
}

void MainWindow::onScanRequested(const QString &path)
{
    clearBanner();
    m_pendingPath = path;
    m_lastPath = path;
    setStatus(tr("Checking %1…").arg(path));
    m_inspectId = connectCall(m_engine, QStringLiteral("inspectPath"),
                              QJsonObject{{QStringLiteral("path"), path}}, this,
                              [this](quint64 id, bool ok, const QJsonObject &res, const QJsonObject &err) {
                                  onInspect(id, ok, res, err);
                              });
}

void MainWindow::onInspect(quint64, bool ok, const QJsonObject &result, const QJsonObject &error)
{
    if (!ok) {
        showBanner(tr("Not a Minecraft instance: %1").arg(error.value(QStringLiteral("message")).toString()));
        setStatus(tr("Invalid path"));
        return;
    }
    const QString path = result.value(QStringLiteral("path")).toString(m_pendingPath);
    m_stack->setCurrentWidget(m_mainView);
    m_progress->setVisible(true);
    m_cancelBtn->setVisible(true);
    m_progress->setRange(0, 0);
    setStatus(tr("Scanning %1…").arg(path));
    m_scanRequestId = connectCall(m_engine, QStringLiteral("scan"),
                                  QJsonObject{{QStringLiteral("path"), path}}, this,
                                  [this](quint64 id, bool ok2, const QJsonObject &res, const QJsonObject &err) {
                                      onScanDone(id, ok2, res, err);
                                  });
}

void MainWindow::onScanProgress(quint64, const QString &, const QJsonObject &data)
{
    const int done = data.value(QStringLiteral("done")).toInt();
    const int total = data.value(QStringLiteral("total")).toInt();
    if (total > 0) {
        m_progress->setRange(0, total);
        m_progress->setValue(done);
    }
    setStatus(total > 0 ? tr("Scanning… %1/%2").arg(done).arg(total) : tr("Scanning…"));
}

void MainWindow::onScanDone(quint64, bool ok, const QJsonObject &result, const QJsonObject &error)
{
    m_progress->setVisible(false);
    m_cancelBtn->setVisible(false);
    if (!ok) {
        showBanner(tr("Scan failed: %1").arg(error.value(QStringLiteral("message")).toString()));
        setStatus(tr("Scan failed"));
        return;
    }
    const QString scanId = result.value(QStringLiteral("scanId")).toString();
    m_engineScanId = scanId;
    m_details->clearDetails();
    m_viewCtl->setScanId(scanId);
    m_sortField.clear();
    m_sortDesc = false;
    m_sortActive = false;
    auto opts = m_viewCtl->options();
    opts.search.clear();
    opts.loader.clear();
    opts.side.clear();
    opts.author.clear();
    opts.sort.clear();
    opts.groupBy.clear();
    opts.showDisabled = m_disabledCheck->isChecked();
    opts.showNested = m_nestedCheck->isChecked();
    opts.unrecognisedOnly = m_unrecCheck->isChecked();
    m_viewCtl->setOptions(opts);
    m_search->clear();
    m_sideCombo->setCurrentIndex(0);
    m_groupCombo->setCurrentIndex(0);
    statusBar()->showMessage(tr("Done"));
}

void MainWindow::onStateReady(const QJsonObject &state)
{
    m_model->setState(state);
    updateHeader();
    // Loader chips come from the unfiltered options and stay stable under
    // filtering; keep the current selection if it still exists.
    const QString current = m_viewCtl->options().loader;
    m_chipBar->setOptions(m_model->loaderOptions());
    if (!current.isEmpty() && !m_model->loaderOptions().contains(current))
        m_chipBar->setSelected(QString());
    else
        m_chipBar->setSelected(current);
    setStatus(tr("%1 mod(s)").arg(m_model->total()));
}

void MainWindow::updateHeader()
{
    QString text = m_model->instanceName();
    if (!m_model->loadersText().isEmpty())
        text += QStringLiteral(" — ") + m_model->loadersText();
    text += QStringLiteral(" — ") + tr("%1 mod(s)").arg(m_model->total());
    if (m_model->disabled() > 0)
        text += tr(" (%1 disabled hidden)").arg(m_model->disabled());
    if (m_model->warningCount() > 0)
        text += tr(" (%1 warning(s))").arg(m_model->warningCount());
    m_headerLabel->setText(text);
}

void MainWindow::onViewError(const QString &code, const QString &message)
{
    setStatus(tr("View error (%1): %2").arg(code, message));
}

void MainWindow::onSortRequested(int column)
{
    if (column < 0 || column >= 8)
        return;
    const QString field = QString::fromLatin1(kSortFields[column]);
    QString spec;
    if (m_sortActive && m_sortField == field) {
        if (!m_sortDesc) {
            m_sortDesc = true;
            spec = QLatin1Char('-') + field;
        } else {
            // Third click clears the sort.
            m_sortActive = false;
            m_sortField.clear();
            m_sortDesc = false;
        }
    } else {
        m_sortActive = true;
        m_sortField = field;
        m_sortDesc = false;
        spec = field;
    }
    auto opts = m_viewCtl->options();
    opts.sort = spec;
    m_viewCtl->setOptions(opts);
}

void MainWindow::refreshView()
{
    auto opts = m_viewCtl->options();
    opts.side = m_sideCombo->currentData().toString();
    opts.groupBy = m_groupCombo->currentData().toString();
    opts.showDisabled = m_disabledCheck->isChecked();
    opts.showNested = m_nestedCheck->isChecked();
    opts.unrecognisedOnly = m_unrecCheck->isChecked();
    m_viewCtl->setOptions(opts);
}

void MainWindow::onSearchEdited()
{
    m_searchDebounce->start();
}

void MainWindow::applySearchNow()
{
    auto opts = m_viewCtl->options();
    opts.search = m_search->text();
    m_viewCtl->setOptions(opts);
}

void MainWindow::onCancelScan()
{
    if (m_scanRequestId != 0)
        m_engine->call(QStringLiteral("cancel"), QJsonObject{{QStringLiteral("id"), static_cast<double>(m_scanRequestId)}});
}

void MainWindow::onSelectionChanged()
{
    const QModelIndex idx = m_table->currentIndex();
    const QString rowId = m_model->rowId(idx.row());
    if (rowId.isEmpty() || m_engineScanId.isEmpty()) {
        m_details->clearDetails();
        return;
    }
    ++m_detailsSeq;
    const quint64 seq = m_detailsSeq;
    connectCall(m_engine, QStringLiteral("details"),
                QJsonObject{{QStringLiteral("scanId"), m_engineScanId},
                            {QStringLiteral("rowId"), rowId}}, this,
                [this, seq](quint64, bool ok, const QJsonObject &res, const QJsonObject &err) {
                    onDetails(seq, ok, res, err);
                });
}

void MainWindow::onDetails(quint64 seq, bool ok, const QJsonObject &result, const QJsonObject &error)
{
    if (seq != m_detailsSeq)
        return; // stale details response
    if (!ok) {
        setStatus(tr("Details failed: %1").arg(error.value(QStringLiteral("message")).toString()));
        return;
    }
    m_details->setDetails(result);
}

void MainWindow::onCheck()
{
    if (m_engineScanId.isEmpty())
        return;
    connectCall(m_engine, QStringLiteral("check"),
                QJsonObject{{QStringLiteral("scanId"), m_engineScanId}}, this,
                [this](quint64, bool ok, const QJsonObject &res, const QJsonObject &err) {
                    onCheckDone(0, ok, res, err);
                });
}

void MainWindow::onCheckDone(quint64, bool ok, const QJsonObject &result, const QJsonObject &error)
{
    if (!ok) {
        QMessageBox::warning(this, tr("Check"), error.value(QStringLiteral("message")).toString());
        return;
    }
    ReportDialog::showIssues(this, result.value(QStringLiteral("issues")).toArray());
}

void MainWindow::onCompare()
{
    if (m_engineScanId.isEmpty())
        return;
    const QString path = QFileDialog::getExistingDirectory(this, tr("Select instance to compare with"));
    if (path.isEmpty())
        return;
    setStatus(tr("Comparing with %1…").arg(path));
    connectCall(m_engine, QStringLiteral("compare"),
                QJsonObject{{QStringLiteral("scanId"), m_engineScanId},
                            {QStringLiteral("path"), path}}, this,
                [this](quint64, bool ok, const QJsonObject &res, const QJsonObject &err) {
                    onCompareDone(0, ok, res, err);
                });
}

void MainWindow::onCompareDone(quint64, bool ok, const QJsonObject &result, const QJsonObject &error)
{
    if (!ok) {
        QMessageBox::warning(this, tr("Compare"), error.value(QStringLiteral("message")).toString());
        setStatus(tr("Compare failed"));
        return;
    }
    // The scan's current view is now the diff; refresh the table and fetch
    // the diff rows for the report dialog.
    m_viewCtl->refresh();
    const QString scanId = result.value(QStringLiteral("scanId")).toString(m_engineScanId);
    m_compareViewId = connectCall(m_engine, QStringLiteral("view"),
                                  QJsonObject{
                                      {QStringLiteral("scanId"), scanId},
                                      {QStringLiteral("view"), QJsonObject{
                                           {QStringLiteral("showDisabled"), true},
                                           {QStringLiteral("showNested"), true},
                                       }}},
                                  this,
                                  [this](quint64, bool ok2, const QJsonObject &res, const QJsonObject &err) {
                                      onCompareView(0, ok2, res, err);
                                  });
}

void MainWindow::onCompareView(quint64, bool ok, const QJsonObject &result, const QJsonObject &error)
{
    Q_UNUSED(m_compareViewId);
    if (!ok) {
        setStatus(tr("Compare view failed: %1").arg(error.value(QStringLiteral("message")).toString()));
        return;
    }
    const QJsonObject state = result.value(QStringLiteral("state")).toObject();
    const QJsonObject summary{
        {QStringLiteral("instance"), state.value(QStringLiteral("instance"))},
        {QStringLiteral("total"), state.value(QStringLiteral("total"))}};
    ReportDialog::showDiff(this, summary, state.value(QStringLiteral("rows")).toArray());
    setStatus(tr("Compare done"));
}

void MainWindow::onCopyMarkdown()
{
    if (m_engineScanId.isEmpty())
        return;
    connectCall(m_engine, QStringLiteral("markdown"),
                QJsonObject{{QStringLiteral("scanId"), m_engineScanId},
                            {QStringLiteral("view"), QJsonObject{}}}, this,
                [this](quint64, bool ok, const QJsonObject &res, const QJsonObject &err) {
                    onMarkdownDone(0, ok, res, err);
                });
}

void MainWindow::onMarkdownDone(quint64, bool ok, const QJsonObject &result, const QJsonObject &error)
{
    if (!ok) {
        QMessageBox::warning(this, tr("Copy"), error.value(QStringLiteral("message")).toString());
        return;
    }
    QGuiApplication::clipboard()->setText(result.value(QStringLiteral("content")).toString());
    setStatus(tr("Markdown copied"));
}

void MainWindow::onExport()
{
    if (m_engineScanId.isEmpty())
        return;
    const QString filter = tr("Markdown (*.md);;JSON (*.json);;CSV (*.csv);;Text (*.txt);;HTML (*.html)");
    QString path = QFileDialog::getSaveFileName(this, tr("Export mods"), QString(), filter);
    if (path.isEmpty())
        return;
    QString format = QFileInfo(path).suffix().toLower();
    if (format == QLatin1String("md"))
        format = QStringLiteral("markdown");
    if (format == QLatin1String("txt"))
        format = QStringLiteral("text");
    m_exportPath = path;
    connectCall(m_engine, QStringLiteral("export"),
                QJsonObject{{QStringLiteral("scanId"), m_engineScanId},
                            {QStringLiteral("format"), format},
                            {QStringLiteral("view"), QJsonObject{}}}, this,
                [this, path](quint64, bool ok, const QJsonObject &res, const QJsonObject &err) {
                    onExportDone(0, ok, res, err);
                });
}

void MainWindow::onExportDone(quint64, bool ok, const QJsonObject &result, const QJsonObject &error)
{
    if (!ok) {
        QMessageBox::warning(this, tr("Export"), error.value(QStringLiteral("message")).toString());
        return;
    }
    QSaveFile file(m_exportPath);
    if (!file.open(QIODevice::WriteOnly)) {
        QMessageBox::warning(this, tr("Export"), tr("Cannot write %1").arg(m_exportPath));
        return;
    }
    file.write(result.value(QStringLiteral("content")).toString().toUtf8());
    if (!file.commit()) {
        QMessageBox::warning(this, tr("Export"), tr("Failed to write %1").arg(m_exportPath));
        return;
    }
    setStatus(tr("Exported %1").arg(m_exportPath));
}

void MainWindow::onEngineError(const QString &message)
{
    setStatus(tr("Engine: %1").arg(message));
}

void MainWindow::onEngineExited(int exitCode, QProcess::ExitStatus)
{
    showBanner(tr("Engine exited (code %1). Captured stderr is in the details log.").arg(exitCode));
    setStatus(tr("Engine stopped"));
}

void MainWindow::onProtocolMismatch(const QString &message)
{
    showBanner(tr("Protocol mismatch: %1").arg(message));
}

void MainWindow::onRestart()
{
    m_engine->stop();
    startEngine();
}

void MainWindow::showBanner(const QString &text)
{
    m_bannerText->setText(text);
    m_banner->setVisible(true);
}

void MainWindow::clearBanner()
{
    m_banner->setVisible(false);
}

void MainWindow::setStatus(const QString &text)
{
    m_status->setText(text);
    statusBar()->showMessage(text);
}
