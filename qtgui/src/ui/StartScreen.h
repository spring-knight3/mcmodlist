#pragma once

#include <QWidget>

class QListWidget;
class QLineEdit;
class EngineClient;

// StartScreen lists detected launcher instances and accepts a manual path,
// Browse, or drag-and-drop. It validates paths through inspectPath via the
// scanRequested signal (MainWindow performs the engine calls).
class StartScreen : public QWidget
{
    Q_OBJECT
public:
    explicit StartScreen(QWidget *parent = nullptr);

    void setInstances(const QList<QPair<QString, QString>> &instances);
    void setPath(const QString &path);
    QString path() const;

signals:
    void scanRequested(const QString &path);

protected:
    void dragEnterEvent(QDragEnterEvent *event) override;
    void dropEvent(QDropEvent *event) override;

private:
    void onBrowse();
    void onScan();

    QListWidget *m_list;
    QLineEdit *m_path;
};
