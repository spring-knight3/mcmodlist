#include "ModTableModel.h"

#include <QJsonArray>
#include <QJsonObject>

namespace {

const char *kColumns[] = {"Name", "ID", "Version", "Authors", "Jar", "Loader", "Side", "Status"};

QString field(const QJsonObject &row, int column)
{
    switch (column) {
    case ModTableModel::ColName: return row.value(QStringLiteral("name")).toString();
    case ModTableModel::ColID: return row.value(QStringLiteral("id")).toString();
    case ModTableModel::ColVersion: return row.value(QStringLiteral("version")).toString();
    case ModTableModel::ColAuthors: return row.value(QStringLiteral("authors")).toString();
    case ModTableModel::ColJar: return row.value(QStringLiteral("jar")).toString();
    case ModTableModel::ColLoader: return row.value(QStringLiteral("loader")).toString();
    case ModTableModel::ColSide: return row.value(QStringLiteral("side")).toString();
    case ModTableModel::ColStatus: return row.value(QStringLiteral("status")).toString();
    default: return {};
    }
}

} // namespace

ModTableModel::ModTableModel(QObject *parent)
    : QAbstractTableModel(parent)
{
}

void ModTableModel::setState(const QJsonObject &state)
{
    beginResetModel();
    clear();

    m_instanceName = state.value(QStringLiteral("instance")).toObject()
                         .value(QStringLiteral("name")).toString();
    m_total = state.value(QStringLiteral("total")).toInt();
    m_disabled = state.value(QStringLiteral("disabled")).toInt();
    m_warnings = state.value(QStringLiteral("warnings")).toArray().size();

    QStringList loaderParts;
    const QJsonArray loaders = state.value(QStringLiteral("loaders")).toArray();
    for (const QJsonValue &v : loaders) {
        const QJsonObject o = v.toObject();
        loaderParts << QStringLiteral("%1 (%2)")
                           .arg(o.value(QStringLiteral("label")).toString())
                           .arg(o.value(QStringLiteral("count")).toInt());
    }
    m_loadersText = loaderParts.join(QLatin1Char(' '));

    const QJsonArray opts = state.value(QStringLiteral("loaderOptions")).toArray();
    for (const QJsonValue &v : opts)
        m_loaderOptions << v.toString();

    m_rows = state.value(QStringLiteral("rows")).toArray();
    m_groups = state.value(QStringLiteral("groups")).toArray();
    m_modRowCount = m_rows.size();
    const QJsonArray &groups = m_groups;

    if (groups.isEmpty()) {
        m_entries.reserve(m_modRowCount);
        for (int i = 0; i < m_modRowCount; ++i)
            m_entries.append({false, i, -1});
    } else {
        // Insert a header entry before each group's rows, in engine order.
        int pos = 0;
        for (int g = 0; g < groups.size(); ++g) {
            const QJsonObject group = groups.at(g).toObject();
            const int start = group.value(QStringLiteral("start")).toInt();
            const int count = group.value(QStringLiteral("count")).toInt();
            while (pos < start && pos < m_modRowCount) {
                m_entries.append({false, pos, -1});
                ++pos;
            }
            m_entries.append({true, -1, g});
            for (int i = 0; i < count && pos < m_modRowCount; ++i, ++pos)
                m_entries.append({false, pos, -1});
        }
        while (pos < m_modRowCount) {
            m_entries.append({false, pos, -1});
            ++pos;
        }
    }
    endResetModel();
}

void ModTableModel::clear()
{
    m_entries.clear();
    m_rows = QJsonArray();
    m_groups = QJsonArray();
    m_modRowCount = 0;
    m_instanceName.clear();
    m_loadersText.clear();
    m_total = m_disabled = m_warnings = 0;
    m_loaderOptions.clear();
}

bool ModTableModel::isGroupHeader(int row) const
{
    return row >= 0 && row < m_entries.size() && m_entries.at(row).header;
}

int ModTableModel::modIndex(int row) const
{
    if (row < 0 || row >= m_entries.size() || m_entries.at(row).header)
        return -1;
    return m_entries.at(row).index;
}

QString ModTableModel::rowId(int row) const
{
    const int idx = modIndex(row);
    if (idx < 0)
        return {};
    return m_rows.at(idx).toObject().value(QStringLiteral("rowId")).toString();
}

int ModTableModel::rowCount(const QModelIndex &parent) const
{
    return parent.isValid() ? 0 : m_entries.size();
}

int ModTableModel::columnCount(const QModelIndex &parent) const
{
    return parent.isValid() ? 0 : ColCount;
}

QVariant ModTableModel::data(const QModelIndex &index, int role) const
{
    if (!index.isValid() || index.row() >= m_entries.size() || index.column() >= ColCount)
        return {};
    const Entry &entry = m_entries.at(index.row());
    if (entry.header) {
        if (index.column() == ColName) {
            if (role == Qt::DisplayRole) {
                const QJsonObject group = m_groups.at(entry.group).toObject();
                return group.value(QStringLiteral("title")).toString();
            }
            if (role == Qt::UserRole + 1)
                return true; // marks a group header for the delegate
        }
        return {};
    }
    if (role == Qt::DisplayRole || role == Qt::EditRole) {
        const QJsonObject row = m_rows.at(entry.index).toObject();
        return field(row, index.column());
    }
    if (role == Qt::ToolTipRole && index.column() == ColJar)
        return m_rows.at(entry.index).toObject().value(QStringLiteral("jar")).toString();
    return {};
}

QVariant ModTableModel::headerData(int section, Qt::Orientation orientation, int role) const
{
    if (orientation == Qt::Horizontal && role == Qt::DisplayRole && section < ColCount)
        return QString::fromLatin1(kColumns[section]);
    return {};
}

Qt::ItemFlags ModTableModel::flags(const QModelIndex &index) const
{
    Qt::ItemFlags f = QAbstractTableModel::flags(index);
    if (index.isValid() && isGroupHeader(index.row()))
        f &= ~(Qt::ItemIsSelectable | Qt::ItemIsEditable);
    return f;
}
