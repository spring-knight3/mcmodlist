#include "ModTableView.h"

#include <QHeaderView>
#include <QMouseEvent>
#include <QPainter>
#include <QStyle>
#include <QStyleOptionViewItem>

#include "../model/ModTableModel.h"
#include "PaletteUtil.h"

void ModTableDelegate::paint(QPainter *painter, const QStyleOptionViewItem &option,
                             const QModelIndex &index) const
{
    const bool header = index.data(Qt::UserRole + 1).toBool();
    if (!header) {
        QStyledItemDelegate::paint(painter, option, index);
        return;
    }

    // Group header row: span the whole viewport width.
    int width = option.rect.width();
    if (const auto *view = qobject_cast<const QTableView *>(option.widget))
        width = view->viewport()->width();
    QStyleOptionViewItem opt = option;
    opt.rect = QRect(0, option.rect.top(), width, option.rect.height());
    painter->save();
    painter->fillRect(opt.rect, PaletteUtil::color(QPalette::AlternateBase));

    QStyleOptionViewItem textOpt = opt;
    textOpt.text = index.data(Qt::DisplayRole).toString();
    textOpt.palette.setColor(QPalette::Text, PaletteUtil::color(QPalette::WindowText));
    textOpt.palette.setColor(QPalette::ButtonText, PaletteUtil::color(QPalette::WindowText));
    QFont font = textOpt.font;
    font.setBold(true);
    textOpt.font = font;
    const QWidget *w = option.widget;
    w->style()->drawControl(QStyle::CE_ItemViewItem, &textOpt, painter, w);
    painter->restore();
}

ModTableView::ModTableView(QWidget *parent)
    : QTableView(parent)
{
    setItemDelegate(new ModTableDelegate(this));
    setSelectionBehavior(QAbstractItemView::SelectRows);
    setSelectionMode(QAbstractItemView::SingleSelection);
    setAlternatingRowColors(true);
    setSortingEnabled(false); // engine order is authoritative
    horizontalHeader()->setSectionsClickable(true);
    connect(horizontalHeader(), &QHeaderView::sectionClicked,
            this, &ModTableView::onHeaderClicked);
    verticalHeader()->setVisible(false);
}

void ModTableView::onHeaderClicked(int column)
{
    emit sortRequested(column);
}

void ModTableView::mouseDoubleClickEvent(QMouseEvent *event)
{
    // Double-click does nothing; details arrive in R4.
    Q_UNUSED(event);
}
