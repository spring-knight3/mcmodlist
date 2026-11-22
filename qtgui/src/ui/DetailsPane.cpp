#include "DetailsPane.h"

#include <QClipboard>
#include <QDesktopServices>
#include <QGuiApplication>
#include <QEvent>
#include <QJsonArray>
#include <QJsonObject>
#include <QLabel>
#include <QPushButton>
#include <QUrl>
#include <QVBoxLayout>

#include "PaletteUtil.h"

DetailsPane::DetailsPane(QWidget *parent)
    : QWidget(parent)
    , m_layout(new QVBoxLayout(this))
    , m_title(new QLabel(this))
    , m_description(new QLabel(this))
    , m_fields(new QWidget(this))
    , m_fieldsLayout(new QVBoxLayout(m_fields))
    , m_warnings(new QLabel(this))
{
    m_layout->setContentsMargins(8, 8, 8, 8);
    m_title->setWordWrap(true);
    QFont f = m_title->font();
    f.setBold(true);
    m_title->setFont(f);
    m_description->setWordWrap(true);
    m_warnings->setWordWrap(true);
    m_warnings->setTextInteractionFlags(Qt::TextBrowserInteraction);

    m_layout->addWidget(m_title);
    m_layout->addWidget(m_description);
    m_layout->addWidget(m_fields);
    m_layout->addWidget(m_warnings);
    m_layout->addStretch(1);
    clearDetails();
}

void DetailsPane::clearDetails()
{
    m_title->setText(tr("Select a mod for details."));
    m_description->clear();
    while (QLayoutItem *item = m_fieldsLayout->takeAt(0)) {
        delete item->widget();
        delete item;
    }
    m_warnings->clear();
    m_warningsText.clear();
    m_hasWarnings = false;
}

void DetailsPane::setDetails(const QJsonObject &details)
{
    const QJsonObject mod = details.value(QStringLiteral("mod")).toObject();
    if (mod.isEmpty()) {
        clearDetails();
        return;
    }
    clearDetails();
    const QString name = mod.value(QStringLiteral("name")).toString();
    const QString id = mod.value(QStringLiteral("id")).toString();
    m_title->setText(name.isEmpty() ? id : name + QStringLiteral(" (") + id + QLatin1Char(')'));
    m_description->setText(mod.value(QStringLiteral("description")).toString());

    addField(tr("Version:"), mod.value(QStringLiteral("version")).toString());
    addField(tr("Authors:"), mod.value(QStringLiteral("authors")).toString());
    addField(tr("Loader:"), mod.value(QStringLiteral("loader")).toString()
                                 + QStringLiteral(" · ") + mod.value(QStringLiteral("side")).toString()
                                 + QStringLiteral(" · ") + mod.value(QStringLiteral("status")).toString());
    addField(tr("License:"), mod.value(QStringLiteral("license")).toString());

    const QString jar = mod.value(QStringLiteral("jar")).toString();
    if (!jar.isEmpty()) {
        auto *row = new QWidget(m_fields);
        auto *rowLayout = new QHBoxLayout(row);
        rowLayout->setContentsMargins(0, 0, 0, 0);
        auto *jarLabel = new QLabel(tr("Jar: ") + jar, row);
        jarLabel->setTextInteractionFlags(Qt::TextSelectableByMouse);
        auto *copy = new QPushButton(tr("Copy path"), row);
        connect(copy, &QPushButton::clicked, this, [jar]() {
            QGuiApplication::clipboard()->setText(jar);
        });
        rowLayout->addWidget(jarLabel, 1);
        rowLayout->addWidget(copy);
        m_fieldsLayout->addWidget(row);
    }

    const QJsonObject links = mod.value(QStringLiteral("links")).toObject();
    for (auto it = links.begin(); it != links.end(); ++it) {
        const QString label = it.key();
        const QUrl url = QUrl::fromUserInput(it.value().toString());
        auto *link = new QLabel(QStringLiteral("<a href=\"%1\">%2</a>").arg(url.toString(), label), m_fields);
        link->setOpenExternalLinks(true);
        link->setTextInteractionFlags(Qt::TextBrowserInteraction);
        m_fieldsLayout->addWidget(link);
    }

    QStringList warnings;
    const QJsonArray arr = details.value(QStringLiteral("warnings")).toArray();
    for (const QJsonValue &v : arr) {
        const QJsonObject w = v.toObject();
        warnings << QStringLiteral("⚠ %1: %2")
                        .arg(w.value(QStringLiteral("type")).toString(),
                             w.value(QStringLiteral("message")).toString());
    }
    m_warningsText = warnings.join(QLatin1Char('\n'));
    m_hasWarnings = !warnings.isEmpty();
    applyWarningPalette();
}

bool DetailsPane::event(QEvent *event)
{
    if (event->type() == QEvent::PaletteChange || event->type() == QEvent::ApplicationPaletteChange)
        applyWarningPalette();
    return QWidget::event(event);
}

void DetailsPane::applyWarningPalette()
{
    m_warnings->setText(m_warningsText);
    if (m_hasWarnings) {
        QPalette pal = m_warnings->palette();
        pal.setColor(QPalette::Text, PaletteUtil::color(QPalette::BrightText));
        m_warnings->setPalette(pal);
    }
}

void DetailsPane::addField(const QString &label, const QString &value)
{
    if (value.trimmed().isEmpty())
        return;
    auto *row = new QHBoxLayout;
    auto *name = new QLabel(label, m_fields);
    name->setMinimumWidth(70);
    auto *val = new QLabel(value, m_fields);
    val->setWordWrap(true);
    val->setTextInteractionFlags(Qt::TextSelectableByMouse);
    row->addWidget(name);
    row->addWidget(val, 1);
    m_fieldsLayout->addLayout(row);
}
