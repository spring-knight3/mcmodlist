#include <QApplication>
#include <QCheckBox>
#include <QDir>
#include <QFile>
#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QPalette>
#include <QPushButton>
#include <QSignalSpy>
#include <QStyle>
#include <QTest>
#include <QTemporaryDir>

#include "engine/EngineClient.h"
#include "engine/EngineLocator.h"
#include "model/ModTableModel.h"
#include "model/ViewController.h"
#include "ui/ChipBar.h"
#include "ui/PaletteUtil.h"

// --- Minimal stored (uncompressed) ZIP writer for fixture jars ---

static quint32 crc32(const QByteArray &data)
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

static void append16(QByteArray &out, quint16 v)
{
    out.append(static_cast<char>(v & 0xFF));
    out.append(static_cast<char>((v >> 8) & 0xFF));
}

static void append32(QByteArray &out, quint32 v)
{
    for (int i = 0; i < 4; ++i)
        out.append(static_cast<char>((v >> (8 * i)) & 0xFF));
}

static QByteArray buildJar(const QList<QPair<QString, QByteArray>> &entries)
{
    QByteArray out;
    QList<quint32> offsets;
    for (const auto &[name, data] : entries) {
        const QByteArray nameBytes = name.toUtf8();
        const quint32 crc = crc32(data);
        offsets.append(static_cast<quint32>(out.size()));
        append32(out, 0x04034b50); // local file header
        append16(out, 20);
        append16(out, 0);
        append16(out, 0); // stored
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
        append32(out, 0x02014b50); // central directory header
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
    append32(out, 0x06054b50); // end of central directory
    append16(out, 0);
    append16(out, 0);
    append16(out, entries.size());
    append16(out, entries.size());
    append32(out, cdSize);
    append32(out, cdStart);
    append16(out, 0);
    return out;
}

static QString writeFile(const QString &path, const QByteArray &data)
{
    QFile f(path);
    if (!f.open(QIODevice::WriteOnly))
        return {};
    f.write(data);
    return path;
}

static QString makeInstance(QTemporaryDir &dir, int bulkMods)
{
    const QString root = dir.filePath(QStringLiteral("instance"));
    QDir().mkpath(root + QStringLiteral("/mods"));

    const QByteArray fabric = QByteArrayLiteral(
        "{\"id\":\"fab\",\"name\":\"Fab Mod\",\"version\":\"1.0.0\",\"authors\":[\"Alice\"]}");
    writeFile(root + QStringLiteral("/mods/fabric.jar"),
              buildJar({{QStringLiteral("fabric.mod.json"), fabric}}));

    QString toml = QStringLiteral("modLoader = \"javafml\"\n");
    for (int i = 0; i < bulkMods; ++i)
        toml += QStringLiteral("[[mods]]\nmodId = \"bulk-%1\"\nversion = \"1\"\ndisplayName = \"Bulk %1\"\n")
                    .arg(i, 3, 10, QLatin1Char('0'));
    writeFile(root + QStringLiteral("/mods/bulk.jar"),
              buildJar({{QStringLiteral("META-INF/mods.toml"), toml.toUtf8()}}));
    return root;
}

static QJsonObject callSync(EngineClient &client, const QString &method, const QJsonObject &params,
                            QSignalSpy &spy)
{
    client.call(method, params);
    for (int i = 0; i < 300 && spy.count() < 1; ++i)
        QTest::qWait(50);
    if (spy.isEmpty())
        return {};
    const auto args = spy.takeFirst();
    if (!args.at(1).toBool())
        return {};
    return args.at(2).toJsonObject();
}

class MainViewTest : public QObject
{
    Q_OBJECT
private slots:
    void modelMatchesEngineState();
    void groupedModelBoundaries();
    void latestWins();
    void chipBarToggles();
    void paletteSwap();
};

void MainViewTest::modelMatchesEngineState()
{
    QTemporaryDir dir;
    QVERIFY(dir.isValid());
    const QString root = makeInstance(dir, 300);
    const QString enginePath = findEnginePath();
    if (enginePath.isEmpty())
        QSKIP("mcmodlist-engine not found");

    EngineClient client;
    client.setEnginePath(enginePath);
    QVERIFY(client.start());
    QSignalSpy spy(&client, &EngineClient::response);

    const QJsonObject scanRes = callSync(client, QStringLiteral("scan"),
                                         QJsonObject{{QStringLiteral("path"), root}}, spy);
    const QString scanId = scanRes.value(QStringLiteral("scanId")).toString();
    QVERIFY(!scanId.isEmpty());

    const QJsonObject viewRes = callSync(client, QStringLiteral("view"),
                                         QJsonObject{
                                             {QStringLiteral("scanId"), scanId},
                                             {QStringLiteral("view"), QJsonObject{
                                                  {QStringLiteral("groupBy"), QString()},
                                                  {QStringLiteral("showDisabled"), true},
                                              }}},
                                         spy);
    const QJsonObject state = viewRes.value(QStringLiteral("state")).toObject();
    const QJsonArray rows = state.value(QStringLiteral("rows")).toArray();
    QCOMPARE(rows.size(), 301);

    ModTableModel model;
    model.setState(state);
    QCOMPARE(model.rowCount(), rows.size());
    QCOMPARE(model.modRowCount(), rows.size());
    QCOMPARE(model.total(), 301);

    // Rows appear in engine order with no reordering.
    for (int i = 0; i < rows.size(); ++i) {
        const QJsonObject row = rows.at(i).toObject();
        const QModelIndex idx = model.index(i, ModTableModel::ColName);
        QCOMPARE(idx.data().toString(), row.value(QStringLiteral("name")).toString());
        QCOMPARE(model.modIndex(i), i);
    }
    QVERIFY(model.loaderOptions().contains(QStringLiteral("fabric")));
    QVERIFY(model.loaderOptions().contains(QStringLiteral("forge")));
    QVERIFY(model.loadersText().contains(QStringLiteral("fabric")));
    QVERIFY(model.instanceName() == state.value(QStringLiteral("instance")).toObject()
                                          .value(QStringLiteral("name")).toString());
    client.stop();
}

void MainViewTest::groupedModelBoundaries()
{
    QTemporaryDir dir;
    QVERIFY(dir.isValid());
    const QString root = makeInstance(dir, 5);
    const QString enginePath = findEnginePath();
    if (enginePath.isEmpty())
        QSKIP("mcmodlist-engine not found");

    EngineClient client;
    client.setEnginePath(enginePath);
    QVERIFY(client.start());
    QSignalSpy spy(&client, &EngineClient::response);

    const QJsonObject scanRes = callSync(client, QStringLiteral("scan"),
                                         QJsonObject{{QStringLiteral("path"), root}}, spy);
    const QString scanId = scanRes.value(QStringLiteral("scanId")).toString();
    QVERIFY(!scanId.isEmpty());

    const QJsonObject viewRes = callSync(client, QStringLiteral("view"),
                                         QJsonObject{
                                             {QStringLiteral("scanId"), scanId},
                                             {QStringLiteral("view"), QJsonObject{
                                                  {QStringLiteral("groupBy"), QStringLiteral("loader")},
                                              }}},
                                         spy);
    const QJsonObject state = viewRes.value(QStringLiteral("state")).toObject();
    const QJsonArray rows = state.value(QStringLiteral("rows")).toArray();
    const QJsonArray groups = state.value(QStringLiteral("groups")).toArray();
    QVERIFY(groups.size() >= 2);

    ModTableModel model;
    model.setState(state);
    QCOMPARE(model.rowCount(), rows.size() + groups.size());

    int pos = 0;
    for (int g = 0; g < groups.size(); ++g) {
        const QJsonObject group = groups.at(g).toObject();
        const int start = group.value(QStringLiteral("start")).toInt();
        const int count = group.value(QStringLiteral("count")).toInt();
        const QString title = group.value(QStringLiteral("title")).toString();

        QVERIFY(model.isGroupHeader(pos));
        QCOMPARE(model.index(pos, ModTableModel::ColName).data().toString(), title);
        QVERIFY(!(model.flags(model.index(pos, 0)) & Qt::ItemIsSelectable));

        for (int i = 0; i < count; ++i) {
            ++pos;
            QVERIFY(!model.isGroupHeader(pos));
            const int modIdx = model.modIndex(pos);
            QCOMPARE(modIdx, start + i);
            QCOMPARE(model.index(pos, ModTableModel::ColName).data().toString(),
                     rows.at(start + i).toObject().value(QStringLiteral("name")).toString());
        }
        ++pos;
    }
    QCOMPARE(pos, model.rowCount());
    client.stop();
}

void MainViewTest::latestWins()
{
    const QString stub = QStringLiteral(MCMODLIST_QT_TESTS_DIR) + QStringLiteral("/stub-engine.sh");
    EngineClient client;
    client.setEnginePath(stub);
    QVERIFY(client.start());

    ViewController ctl(&client);
    ctl.setScanId(QStringLiteral("scan-1"));
    QSignalSpy spy(&ctl, &ViewController::stateReady);

    ctl.setOptions({});   // request id 1
    ctl.refresh();        // request id 2 (supersedes)
    QTRY_VERIFY_WITH_TIMEOUT(spy.count() >= 1, 5000);
    QTest::qWait(600);    // let the stale response arrive; it must be dropped
    QCOMPARE(spy.count(), 1);
    const QJsonObject state = spy.takeFirst().at(0).toJsonObject();
    QCOMPARE(state.value(QStringLiteral("marker")).toInt(), 2);
    client.stop();
}

void MainViewTest::chipBarToggles()
{
    ChipBar bar;
    bar.setOptions({QStringLiteral("fabric"), QStringLiteral("forge")});
    auto buttons = bar.findChildren<QPushButton *>();
    QCOMPARE(buttons.size(), 3); // All + 2 loaders

    QSignalSpy spy(&bar, &ChipBar::selectionChanged);
    auto findBtn = [&bar](const QString &text) {
        for (QPushButton *b : bar.findChildren<QPushButton *>())
            if (b->text() == text)
                return b;
        return static_cast<QPushButton *>(nullptr);
    };
    QPushButton *fabric = findBtn(QStringLiteral("fabric"));
    QVERIFY(fabric);
    fabric->click();
    QTRY_VERIFY_WITH_TIMEOUT(spy.count() >= 1, 1000);
    QCOMPARE(spy.takeFirst().at(0).toString(), QStringLiteral("fabric"));
    QCOMPARE(bar.selected(), QStringLiteral("fabric"));

    fabric = findBtn(QStringLiteral("fabric"));
    fabric->click(); // toggles off
    QTRY_VERIFY_WITH_TIMEOUT(spy.count() >= 1, 1000);
    QCOMPARE(spy.takeFirst().at(0).toString(), QString());
    QCOMPARE(bar.selected(), QString());
}

void MainViewTest::paletteSwap()
{
    PaletteUtil::invalidate();
    QPalette pal = qApp->palette();
    pal.setColor(QPalette::Highlight, QColor(Qt::red));
    pal.setColor(QPalette::AlternateBase, QColor(Qt::blue));
    qApp->setPalette(pal);
    PaletteUtil::invalidate();
    QVERIFY(PaletteUtil::color(QPalette::Highlight) == QColor(Qt::red));
    QVERIFY(PaletteUtil::color(QPalette::AlternateBase) == QColor(Qt::blue));
    // Restore a sane palette for other tests.
    qApp->setPalette(QApplication::style()->standardPalette());
    PaletteUtil::invalidate();
}

QTEST_MAIN(MainViewTest)
#include "mainview_test.moc"
