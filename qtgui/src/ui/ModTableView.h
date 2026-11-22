#pragma once

#include <QStyledItemDelegate>
#include <QTableView>

class ModTableModel;

// Delegate that draws group header rows across the full table width using
// palette roles only (AlternateBase background, WindowText text).
class ModTableDelegate : public QStyledItemDelegate
{
    Q_OBJECT
public:
    using QStyledItemDelegate::QStyledItemDelegate;

    void paint(QPainter *painter, const QStyleOptionViewItem &option,
               const QModelIndex &index) const override;
};

// ModTableView shows engine rows in engine order. Header clicks are forwarded
// as sortRequested(column); the model never reorders.
class ModTableView : public QTableView
{
    Q_OBJECT
public:
    explicit ModTableView(QWidget *parent = nullptr);

signals:
    void sortRequested(int column);

protected:
    void mouseDoubleClickEvent(QMouseEvent *event) override;

private slots:
    void onHeaderClicked(int column);
};
