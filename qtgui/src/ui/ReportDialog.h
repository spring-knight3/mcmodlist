#pragma once

#include <QDialog>
#include <QJsonArray>
#include <QJsonObject>
#include <QStringList>

class QPlainTextEdit;

// ReportDialog shows Check issues or a Compare diff as plain text.
class ReportDialog : public QDialog
{
    Q_OBJECT
public:
    explicit ReportDialog(const QString &title, const QStringList &lines, QWidget *parent = nullptr);

    static void showIssues(QWidget *parent, const QJsonArray &issues);
    static void showDiff(QWidget *parent, const QJsonObject &summary, const QJsonArray &rows);

private:
    QPlainTextEdit *m_text;
};
