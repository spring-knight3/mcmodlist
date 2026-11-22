#include "ChipBar.h"

#include <QPushButton>

ChipBar::ChipBar(QWidget *parent)
    : QWidget(parent)
    , m_layout(new QHBoxLayout(this))
{
    m_layout->setContentsMargins(0, 0, 0, 0);
    m_layout->setSpacing(4);
    setOptions({});
}

void ChipBar::setOptions(const QStringList &options)
{
    m_options = options;
    rebuild();
}

void ChipBar::setSelected(const QString &label)
{
    m_selected = label;
    if (m_all)
        m_all->setChecked(label.isEmpty());
    for (int i = 0; i < m_buttons.size(); ++i) {
        const QString opt = m_options.value(i);
        m_buttons.at(i)->setChecked(opt == label);
    }
}

void ChipBar::rebuild()
{
    while (QLayoutItem *item = m_layout->takeAt(0)) {
        delete item->widget();
        delete item;
    }
    m_buttons.clear();
    m_all = nullptr;

    auto *all = new QPushButton(tr("All"), this);
    all->setAccessibleName(tr("All loaders"));
    all->setCheckable(true);
    all->setChecked(m_selected.isEmpty());
    m_layout->addWidget(all);
    m_all = all;
    connect(all, &QPushButton::clicked, this, [this]() {
        setSelected(QString());
        emit selectionChanged(QString());
    });

    for (const QString &opt : m_options) {
        auto *btn = new QPushButton(opt, this);
        btn->setAccessibleName(tr("Filter by loader %1").arg(opt));
        btn->setCheckable(true);
        btn->setChecked(opt == m_selected);
        m_layout->addWidget(btn);
        m_buttons.append(btn);
        connect(btn, &QPushButton::clicked, this, [this, opt]() {
            const QString next = (m_selected == opt) ? QString() : opt;
            setSelected(next);
            emit selectionChanged(next);
        });
    }
    m_layout->addStretch(1);
}
