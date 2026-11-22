#include "ReportDialog.h"

#include <QDialogButtonBox>
#include <QMap>
#include <QJsonArray>
#include <QJsonObject>
#include <QPlainTextEdit>
#include <QVBoxLayout>

ReportDialog::ReportDialog(const QString &title, const QStringList &lines, QWidget *parent)
    : QDialog(parent)
    , m_text(new QPlainTextEdit(this))
{
    setWindowTitle(title);
    resize(560, 420);
    m_text->setReadOnly(true);
    m_text->setPlainText(lines.join(QLatin1Char('\n')));
    auto *buttons = new QDialogButtonBox(QDialogButtonBox::Close, this);
    connect(buttons, &QDialogButtonBox::rejected, this, &QDialog::reject);
    connect(buttons, &QDialogButtonBox::accepted, this, &QDialog::accept);
    auto *layout = new QVBoxLayout(this);
    layout->addWidget(m_text);
    layout->addWidget(buttons);
}

void ReportDialog::showIssues(QWidget *parent, const QJsonArray &issues)
{
    QStringList lines;
    if (issues.isEmpty()) {
        lines << QObject::tr("No issues found.");
    } else {
        for (const QJsonValue &v : issues) {
            const QJsonObject issue = v.toObject();
            lines << QStringLiteral("%1 — %2: %3")
                         .arg(issue.value(QStringLiteral("type")).toString(),
                              issue.value(QStringLiteral("mod")).toString(),
                              issue.value(QStringLiteral("message")).toString());
        }
    }
    ReportDialog dlg(tr("Check"), lines, parent);
    dlg.exec();
}

void ReportDialog::showDiff(QWidget *parent, const QJsonObject &summary, const QJsonArray &rows)
{
    QStringList lines;
    const QString instance = summary.value(QStringLiteral("instance")).toObject()
                                 .value(QStringLiteral("name")).toString();
    if (!instance.isEmpty())
        lines << QObject::tr("Instance: %1").arg(instance);
    lines << QObject::tr("Total changes: %1").arg(summary.value(QStringLiteral("total")).toInt());
    lines << QString();

    QMap<QString, QStringList> byStatus;
    for (const QJsonValue &v : rows) {
        const QJsonObject row = v.toObject();
        const QString status = row.value(QStringLiteral("status")).toString();
        QString text = row.value(QStringLiteral("name")).toString();
        const QString id = row.value(QStringLiteral("id")).toString();
        if (!id.isEmpty())
            text += QStringLiteral(" (") + id + QLatin1Char(')');
        const QString old = row.value(QStringLiteral("oldVersion")).toString();
        if (!old.isEmpty())
            text += QStringLiteral(" %1 → %2").arg(old, row.value(QStringLiteral("version")).toString());
        byStatus[status].append(text);
    }
    for (auto it = byStatus.begin(); it != byStatus.end(); ++it) {
        lines << QStringLiteral("%1 (%2):").arg(it.key()).arg(it.value().size());
        for (const QString &line : it.value())
            lines << QStringLiteral("  ") + line;
        lines << QString();
    }
    if (byStatus.isEmpty())
        lines << QObject::tr("No differences.");
    ReportDialog dlg(tr("Compare"), lines, parent);
    dlg.exec();
}
