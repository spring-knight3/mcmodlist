#pragma once

// Minimal stored (uncompressed) ZIP writer + instance fixture helpers for
// Qt tests. Shared by actions_test.cpp (mainview_test.cpp has its own copy).

#include <QDir>
#include <QFile>
#include <QList>
#include <QPair>
#include <QString>
#include <QTemporaryDir>

namespace jarfixture {

inline quint32 crc32(const QByteArray &data)
{
    static quint32 table[256];
    static bool init = false;
    if (!init) {
        for (quint32 i = 0; i < 256; ++i) {
            quint32 c = i;
            for (int k = 0; k < 8; ++k)
                c = (c & 1) ? (0xEDB88320u ^ (c >> 1)) : (c >> 1);
            table[i] = c;
        }
        init = true;
    }
    quint32 c = 0xFFFFFFFFu;
    for (char ch : data)
        c = table[(c ^ static_cast<quint8>(ch)) & 0xFF] ^ (c >> 8);
    return c ^ 0xFFFFFFFFu;
}

inline void append16(QByteArray &out, quint16 v)
{
    out.append(static_cast<char>(v & 0xFF));
    out.append(static_cast<char>((v >> 8) & 0xFF));
}

inline void append32(QByteArray &out, quint32 v)
{
    for (int i = 0; i < 4; ++i)
        out.append(static_cast<char>((v >> (8 * i)) & 0xFF));
}

inline QByteArray buildJar(const QList<QPair<QString, QByteArray>> &entries)
{
    QByteArray out;
    QList<quint32> offsets;
    for (const auto &[name, data] : entries) {
        const QByteArray nameBytes = name.toUtf8();
        const quint32 crc = crc32(data);
        offsets.append(static_cast<quint32>(out.size()));
        append32(out, 0x04034b50);
        append16(out, 20);
        append16(out, 0);
        append16(out, 0);
        append16(out, 0);
        append16(out, 0);
        append32(out, crc);
        append32(out, data.size());
        append32(out, data.size());
        append16(out, nameBytes.size());
        append16(out, 0);
        out += nameBytes;
        out += data;
    }
    const quint32 cdStart = out.size();
    for (int i = 0; i < entries.size(); ++i) {
        const auto &[name, data] = entries.at(i);
        const QByteArray nameBytes = name.toUtf8();
        const quint32 crc = crc32(data);
        append32(out, 0x02014b50);
        append16(out, 20);
        append16(out, 20);
        append16(out, 0);
        append16(out, 0);
        append16(out, 0);
        append16(out, 0);
        append32(out, crc);
        append32(out, data.size());
        append32(out, data.size());
        append16(out, nameBytes.size());
        append16(out, 0);
        append16(out, 0);
        append16(out, 0);
        append16(out, 0);
        append32(out, 0);
        append32(out, offsets.at(i));
        out += nameBytes;
    }
    const quint32 cdSize = out.size() - cdStart;
    append32(out, 0x06054b50);
    append16(out, 0);
    append16(out, 0);
    append16(out, entries.size());
    append16(out, entries.size());
    append32(out, cdSize);
    append32(out, cdStart);
    append16(out, 0);
    return out;
}

inline void writeFile(const QString &path, const QByteArray &data)
{
    QFile f(path);
    f.open(QIODevice::WriteOnly);
    f.write(data);
}

inline void addFabricJar(const QString &root, const QString &name, const QString &id,
                         const QString &version, const QString &extra = QString())
{
    const QByteArray meta = QByteArrayLiteral("{\"id\":\"") + id.toUtf8()
        + QByteArrayLiteral("\",\"name\":\"") + name.toUtf8()
        + QByteArrayLiteral("\",\"version\":\"") + version.toUtf8() + QByteArrayLiteral("\"")
        + (extra.isEmpty() ? QByteArray() : QByteArrayLiteral(",") + extra.toUtf8())
        + QByteArrayLiteral("}");
    writeFile(root + QStringLiteral("/mods/") + id + QStringLiteral(".jar"),
              buildJar({{QStringLiteral("fabric.mod.json"), meta}}));
}

} // namespace jarfixture
