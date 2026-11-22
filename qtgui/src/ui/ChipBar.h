#pragma once

#include <QHBoxLayout>
#include <QJsonObject>
#include <QWidget>
#include <QStringList>

class QPushButton;

// ChipBar renders checkable loader chips in the engine-provided order.
// "All" clears the loader filter; clicking a selected chip clears it too.
class ChipBar : public QWidget
{
    Q_OBJECT
public:
    explicit ChipBar(QWidget *parent = nullptr);

    void setOptions(const QStringList &options);
    void setSelected(const QString &label);
    QString selected() const { return m_selected; }

signals:
    void selectionChanged(const QString &label);

private:
    void rebuild();

    QHBoxLayout *m_layout;
    QStringList m_options;
    QString m_selected;
    QPushButton *m_all = nullptr;
    QList<QPushButton *> m_buttons;
};
