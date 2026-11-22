#pragma once

#include <QAbstractTableModel>
#include <QJsonArray>
#include <QJsonObject>
#include <QString>
#include <QVector>

// ModTableModel displays engine view state exactly as returned: rows in
// engine order, group header rows inserted at the engine-provided
// boundaries. It never filters, sorts, or reorders.
class ModTableModel : public QAbstractTableModel
{
    Q_OBJECT
public:
    enum Column {
        ColName = 0,
        ColID,
        ColVersion,
        ColAuthors,
        ColJar,
        ColLoader,
        ColSide,
        ColStatus,
        ColCount
    };

    explicit ModTableModel(QObject *parent = nullptr);

    // Replaces the model contents with a "state" object from a view response.
    void setState(const QJsonObject &state);
    void clear();

    // True when row is a group header (non-selectable, spans the table).
    bool isGroupHeader(int row) const;
    // Index into the engine rows array for a mod row, -1 for headers.
    int modIndex(int row) const;
    int modRowCount() const { return m_modRowCount; }
    // Engine rowId for a mod row (empty for group headers), used by details.
    QString rowId(int row) const;

    // Header-line data derived from the state (engine-computed values only).
    QString instanceName() const { return m_instanceName; }
    QString loadersText() const { return m_loadersText; }
    int total() const { return m_total; }
    int disabled() const { return m_disabled; }
    QStringList loaderOptions() const { return m_loaderOptions; }
    int warningCount() const { return m_warnings; }

    int rowCount(const QModelIndex &parent = QModelIndex()) const override;
    int columnCount(const QModelIndex &parent = QModelIndex()) const override;
    QVariant data(const QModelIndex &index, int role) const override;
    QVariant headerData(int section, Qt::Orientation orientation, int role) const override;
    Qt::ItemFlags flags(const QModelIndex &index) const override;

private:
    struct Entry {
        bool header = false;
        int index = -1;   // mod row index for mod entries
        int group = -1;   // group index for header entries
    };

    QVector<Entry> m_entries;
    QJsonArray m_rows;
    QJsonArray m_groups;
    int m_modRowCount = 0;
    QString m_instanceName;
    QString m_loadersText;
    int m_total = 0;
    int m_disabled = 0;
    int m_warnings = 0;
    QStringList m_loaderOptions;
};
