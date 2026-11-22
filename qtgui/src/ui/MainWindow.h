#pragma once

#include <QJsonObject>
#include <QMainWindow>
#include <QProcess>

class EngineClient;
class StartScreen;
class ViewController;
class ModTableModel;
class ModTableView;
class ChipBar;
class DetailsPane;
class QLabel;
class QPushButton;
class QSplitter;
class QStackedWidget;
class QLineEdit;
class QComboBox;
class QCheckBox;
class QProgressBar;
class QTimer;
class QWidget;

class MainWindow : public QMainWindow
{
    Q_OBJECT
public:
    explicit MainWindow(QWidget *parent = nullptr);

protected:
    bool event(QEvent *event) override;
    void closeEvent(QCloseEvent *event) override;

private slots:
    void startEngine();
    void onHello(quint64 id, bool ok, const QJsonObject &result, const QJsonObject &error);
    void onListInstances(quint64 id, bool ok, const QJsonObject &result, const QJsonObject &error);
    void onInspect(quint64 id, bool ok, const QJsonObject &result, const QJsonObject &error);
    void onScanProgress(quint64 id, const QString &event, const QJsonObject &data);
    void onScanDone(quint64 id, bool ok, const QJsonObject &result, const QJsonObject &error);
    void onScanRequested(const QString &path);
    void onEngineError(const QString &message);
    void onEngineExited(int exitCode, QProcess::ExitStatus status);
    void onRestart();
    void onProtocolMismatch(const QString &message);

    void onStateReady(const QJsonObject &state);
    void onViewError(const QString &code, const QString &message);
    void onSortRequested(int column);
    void onSearchEdited();
    void applySearchNow();
    void onCancelScan();
    void onSelectionChanged();
    void onDetails(quint64 id, bool ok, const QJsonObject &result, const QJsonObject &error);
    void onCheck();
    void onCheckDone(quint64 id, bool ok, const QJsonObject &result, const QJsonObject &error);
    void onCompare();
    void onCompareDone(quint64 id, bool ok, const QJsonObject &result, const QJsonObject &error);
    void onCompareView(quint64 id, bool ok, const QJsonObject &result, const QJsonObject &error);
    void onCopyMarkdown();
    void onMarkdownDone(quint64 id, bool ok, const QJsonObject &result, const QJsonObject &error);
    void onExport();
    void onExportDone(quint64 id, bool ok, const QJsonObject &result, const QJsonObject &error);
    void onBannerDetails();
    void onStderrLine(const QString &line);

private:
    void buildMainView();
    void showBanner(const QString &text);
    void clearBanner();
    void setStatus(const QString &text);
    void refreshView();
    void updateHeader();
    void restoreSettings();
    void saveSettings() const;

    EngineClient *m_engine;
    ViewController *m_viewCtl;
    StartScreen *m_start;
    QStackedWidget *m_stack;
    QWidget *m_mainView;
    QLabel *m_status;
    QWidget *m_banner;
    QLabel *m_bannerText;
    QPushButton *m_bannerRestart;
    QPushButton *m_bannerDetails;
    QStringList m_stderrLines;
    QString m_lastPath;

    ModTableModel *m_model;
    ModTableView *m_table;
    QSplitter *m_splitter;
    QLabel *m_headerLabel;
    QLineEdit *m_search;
    QComboBox *m_sideCombo;
    ChipBar *m_chipBar;
    QCheckBox *m_disabledCheck;
    QCheckBox *m_nestedCheck;
    QCheckBox *m_unrecCheck;
    QComboBox *m_groupCombo;
    QProgressBar *m_progress;
    QPushButton *m_cancelBtn;
    QTimer *m_searchDebounce;
    DetailsPane *m_details;
    QPushButton *m_checkBtn;
    QPushButton *m_compareBtn;
    QPushButton *m_copyBtn;
    QPushButton *m_exportBtn;
    QString m_engineScanId;
    QString m_exportPath;
    quint64 m_detailsSeq = 0;
    quint64 m_compareViewId = 0;

    quint64 m_helloId = 0;
    quint64 m_inspectId = 0;
    quint64 m_scanRequestId = 0;
    QString m_pendingPath;
    QString m_sortField;
    bool m_sortDesc = false;
    bool m_sortActive = false;
};
