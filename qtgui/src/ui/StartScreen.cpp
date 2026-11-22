#include "StartScreen.h"

#include <QDragEnterEvent>
#include <QDropEvent>
#include <QFileDialog>
#include <QHBoxLayout>
#include <QLabel>
#include <QLineEdit>
#include <QListWidget>
#include <QMimeData>
#include <QPushButton>
#include <QVBoxLayout>

StartScreen::StartScreen(QWidget *parent)
    : QWidget(parent)
    , m_list(new QListWidget(this))
    , m_path(new QLineEdit(this))
{
    setAcceptDrops(true);

    auto *title = new QLabel(tr("Select a Minecraft instance"), this);
    auto *hint = new QLabel(tr("Detected launcher instances, or drop a folder / paste a path:"), this);
    hint->setWordWrap(true);

    m_path->setPlaceholderText(tr("/path/to/instance"));
    m_path->setAccessibleName(tr("Instance path"));
    m_list->setAccessibleName(tr("Detected instances"));
    auto *browse = new QPushButton(tr("Browse…"), this);
    browse->setAccessibleName(tr("Browse for instance folder"));
    auto *scan = new QPushButton(tr("Scan"), this);
    scan->setAccessibleName(tr("Scan instance"));
    scan->setDefault(true);

    connect(browse, &QPushButton::clicked, this, &StartScreen::onBrowse);
    connect(scan, &QPushButton::clicked, this, &StartScreen::onScan);
    connect(m_path, &QLineEdit::returnPressed, this, &StartScreen::onScan);
    connect(m_list, &QListWidget::itemActivated, this, [this](QListWidgetItem *item) {
        m_path->setText(item->data(Qt::UserRole).toString());
    });

    auto *pathRow = new QHBoxLayout;
    pathRow->addWidget(m_path, 1);
    pathRow->addWidget(browse);
    pathRow->addWidget(scan);

    auto *layout = new QVBoxLayout(this);
    layout->addWidget(title);
    layout->addWidget(hint);
    layout->addWidget(m_list, 1);
    layout->addLayout(pathRow);
}

void StartScreen::setInstances(const QList<QPair<QString, QString>> &instances)
{
    m_list->clear();
    for (const auto &[name, path] : instances) {
        auto *item = new QListWidgetItem(name + QStringLiteral(" — ") + path, m_list);
        item->setData(Qt::UserRole, path);
    }
}

void StartScreen::setPath(const QString &path)
{
    m_path->setText(path);
}

QString StartScreen::path() const
{
    return m_path->text().trimmed();
}

void StartScreen::dragEnterEvent(QDragEnterEvent *event)
{
    if (event->mimeData()->hasUrls())
        event->acceptProposedAction();
}

void StartScreen::dropEvent(QDropEvent *event)
{
    const QList<QUrl> urls = event->mimeData()->urls();
    if (!urls.isEmpty())
        m_path->setText(urls.first().toLocalFile());
}

void StartScreen::onBrowse()
{
    const QString dir = QFileDialog::getExistingDirectory(this, tr("Select instance folder"), m_path->text());
    if (!dir.isEmpty())
        m_path->setText(dir);
}

void StartScreen::onScan()
{
    const QString p = path();
    if (!p.isEmpty())
        emit scanRequested(p);
}
