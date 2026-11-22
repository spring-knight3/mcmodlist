#include <QDir>
#include <QFile>
#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QMap>
#include <QSignalSpy>
#include <QTemporaryDir>
#include <QTest>

#include "engine/EngineClient.h"
#include "engine/EngineLocator.h"
#include "jarfixture.h"

class ActionsTest : public QObject
{
    Q_OBJECT
private slots:
    void detailsCheckCompareExport();
    void exportMatchesMarkdownMethod();
    void engineErrorsAreReadable();
};

static QJsonObject callSync(EngineClient &client, const QString &method, const QJsonObject &params,
                            QSignalSpy &spy, int timeoutMs = 20000)
{
    client.call(method, params);
    for (int i = 0; i < timeoutMs / 50 && spy.count() < 1; ++i)
        QTest::qWait(50);
    if (spy.isEmpty())
        return {};
    const auto args = spy.takeFirst();
    if (!args.at(1).toBool())
        return {{QStringLiteral("__error"), args.at(3).toJsonObject()}};
    return args.at(2).toJsonObject();
}

static QString makeInstanceA(QTemporaryDir &dir)
{
    const QString root = dir.filePath(QStringLiteral("a"));
    QDir().mkpath(root + QStringLiteral("/mods"));
    jarfixture::addFabricJar(root, QStringLiteral("Alpha Mod"), QStringLiteral("alpha"),
                             QStringLiteral("1.0.0"),
                             QStringLiteral("\"authors\":[\"Alice\"],\"description\":\"Alpha desc\",\"license\":\"MIT\""));
    jarfixture::addFabricJar(root, QStringLiteral("Beta Mod"), QStringLiteral("beta"),
                             QStringLiteral("2.0.0"));
    return root;
}

static QString makeInstanceB(QTemporaryDir &dir)
{
    const QString root = dir.filePath(QStringLiteral("b"));
    QDir().mkpath(root + QStringLiteral("/mods"));
    jarfixture::addFabricJar(root, QStringLiteral("Alpha Mod"), QStringLiteral("alpha"),
                             QStringLiteral("1.1.0")); // version changed
    jarfixture::addFabricJar(root, QStringLiteral("Gamma Mod"), QStringLiteral("gamma"),
                             QStringLiteral("3.0.0")); // added
    // beta removed
    return root;
}

static QString scanPath(EngineClient &client, QSignalSpy &spy, const QString &path)
{
    const QJsonObject res = callSync(client, QStringLiteral("scan"),
                                     QJsonObject{{QStringLiteral("path"), path}}, spy);
    return res.value(QStringLiteral("scanId")).toString();
}

void ActionsTest::detailsCheckCompareExport()
{
    const QString enginePath = findEnginePath();
    if (enginePath.isEmpty())
        QSKIP("mcmodlist-engine not found");

    QTemporaryDir dir;
    QVERIFY(dir.isValid());
    const QString rootA = makeInstanceA(dir);
    const QString rootB = makeInstanceB(dir);

    EngineClient client;
    client.setEnginePath(enginePath);
    QVERIFY(client.start());
    QSignalSpy spy(&client, &EngineClient::response);

    const QString scanA = scanPath(client, spy, rootA);
    QVERIFY(!scanA.isEmpty());
    const QString scanB = scanPath(client, spy, rootB);
    QVERIFY(!scanB.isEmpty());

    // details
    const QJsonObject viewRes = callSync(client, QStringLiteral("view"),
                                         QJsonObject{{QStringLiteral("scanId"), scanA},
                                                     {QStringLiteral("view"), QJsonObject{}}}, spy);
    const QJsonArray rows = viewRes.value(QStringLiteral("state")).toObject()
                                .value(QStringLiteral("rows")).toArray();
    QCOMPARE(rows.size(), 2);
    QString rowId;
    for (const QJsonValue &v : rows) {
        if (v.toObject().value(QStringLiteral("id")).toString() == QLatin1String("alpha"))
            rowId = v.toObject().value(QStringLiteral("rowId")).toString();
    }
    QVERIFY(!rowId.isEmpty());

    const QJsonObject det = callSync(client, QStringLiteral("details"),
                                     QJsonObject{{QStringLiteral("scanId"), scanA},
                                                 {QStringLiteral("rowId"), rowId}}, spy);
    QVERIFY(det.contains(QStringLiteral("mod")));
    const QJsonObject mod = det.value(QStringLiteral("mod")).toObject();
    QCOMPARE(mod.value(QStringLiteral("name")).toString(), QStringLiteral("Alpha Mod"));
    QCOMPARE(mod.value(QStringLiteral("description")).toString(), QStringLiteral("Alpha desc"));
    QCOMPARE(mod.value(QStringLiteral("license")).toString(), QStringLiteral("MIT"));
    QVERIFY(mod.value(QStringLiteral("links")).toObject().isEmpty()
            || !mod.value(QStringLiteral("links")).toObject().isEmpty());

    // check (clean fixture -> issues array present, possibly empty)
    const QJsonObject check = callSync(client, QStringLiteral("check"),
                                       QJsonObject{{QStringLiteral("scanId"), scanA}}, spy);
    QVERIFY(check.contains(QStringLiteral("issues")));

    // compare A -> B: alpha version-changed, beta removed, gamma added
    const QJsonObject cmp = callSync(client, QStringLiteral("compare"),
                                     QJsonObject{{QStringLiteral("scanId"), scanA},
                                                 {QStringLiteral("path"), rootB}}, spy);
    QVERIFY(cmp.contains(QStringLiteral("scanId")));
    QCOMPARE(cmp.value(QStringLiteral("scanId")).toString(), scanA);

    const QJsonObject diffView = callSync(client, QStringLiteral("view"),
                                          QJsonObject{{QStringLiteral("scanId"), scanA},
                                                      {QStringLiteral("view"), QJsonObject{
                                                           {QStringLiteral("showDisabled"), true},
                                                           {QStringLiteral("showNested"), true}}}}, spy);
    const QJsonArray diffRows = diffView.value(QStringLiteral("state")).toObject()
                                    .value(QStringLiteral("rows")).toArray();
    QMap<QString, int> byStatus;
    QString alphaOld;
    for (const QJsonValue &v : diffRows) {
        const QJsonObject row = v.toObject();
        byStatus[row.value(QStringLiteral("status")).toString()]++;
        if (row.value(QStringLiteral("id")).toString() == QLatin1String("alpha"))
            alphaOld = row.value(QStringLiteral("oldVersion")).toString();
    }
    QCOMPARE(byStatus.value(QStringLiteral("added")), 1);
    QCOMPARE(byStatus.value(QStringLiteral("removed")), 1);
    QCOMPARE(byStatus.value(QStringLiteral("version-changed")), 1);
    QCOMPARE(alphaOld, QStringLiteral("1.0.0"));

    // export markdown + json on the diff view
    const QJsonObject md = callSync(client, QStringLiteral("export"),
                                    QJsonObject{{QStringLiteral("scanId"), scanA},
                                                {QStringLiteral("format"), QStringLiteral("markdown")},
                                                {QStringLiteral("view"), QJsonObject{}}}, spy);
    const QString mdContent = md.value(QStringLiteral("content")).toString();
    QVERIFY(mdContent.contains(QStringLiteral("Gamma Mod")));
    QVERIFY(mdContent.contains(QStringLiteral("Alpha Mod")));

    const QJsonObject js = callSync(client, QStringLiteral("export"),
                                    QJsonObject{{QStringLiteral("scanId"), scanA},
                                                {QStringLiteral("format"), QStringLiteral("json")},
                                                {QStringLiteral("view"), QJsonObject{}}}, spy);
    const QJsonDocument doc = QJsonDocument::fromJson(js.value(QStringLiteral("content")).toString().toUtf8());
    QVERIFY(doc.isObject());
    QCOMPARE(doc.object().value(QStringLiteral("mods")).toArray().size(), 3);

    // markdown method returns the same content as export markdown
    const QJsonObject mdMethod = callSync(client, QStringLiteral("markdown"),
                                          QJsonObject{{QStringLiteral("scanId"), scanA},
                                                      {QStringLiteral("view"), QJsonObject{}}}, spy);
    QCOMPARE(mdMethod.value(QStringLiteral("content")).toString(), mdContent);

    client.stop();
}

void ActionsTest::exportMatchesMarkdownMethod()
{
    const QString enginePath = findEnginePath();
    if (enginePath.isEmpty())
        QSKIP("mcmodlist-engine not found");

    QTemporaryDir dir;
    QVERIFY(dir.isValid());
    const QString root = makeInstanceA(dir);

    EngineClient client;
    client.setEnginePath(enginePath);
    QVERIFY(client.start());
    QSignalSpy spy(&client, &EngineClient::response);

    const QString scanId = scanPath(client, spy, root);
    QVERIFY(!scanId.isEmpty());

    const QJsonObject md = callSync(client, QStringLiteral("export"),
                                    QJsonObject{{QStringLiteral("scanId"), scanId},
                                                {QStringLiteral("format"), QStringLiteral("markdown")},
                                                {QStringLiteral("view"), QJsonObject{
                                                     {QStringLiteral("search"), QStringLiteral("alpha")}}}}, spy);
    QVERIFY(md.value(QStringLiteral("content")).toString().contains(QStringLiteral("Alpha Mod")));
    QVERIFY(!md.value(QStringLiteral("content")).toString().contains(QStringLiteral("Beta Mod")));

    const QJsonObject csv = callSync(client, QStringLiteral("export"),
                                     QJsonObject{{QStringLiteral("scanId"), scanId},
                                                 {QStringLiteral("format"), QStringLiteral("csv")},
                                                 {QStringLiteral("view"), QJsonObject{}}}, spy);
    QVERIFY(csv.value(QStringLiteral("content")).toString().contains(QStringLiteral("Name,ID,Version")));

    client.stop();
}

void ActionsTest::engineErrorsAreReadable()
{
    const QString enginePath = findEnginePath();
    if (enginePath.isEmpty())
        QSKIP("mcmodlist-engine not found");

    EngineClient client;
    client.setEnginePath(enginePath);
    QVERIFY(client.start());
    QSignalSpy spy(&client, &EngineClient::response);

    const QJsonObject res = callSync(client, QStringLiteral("view"),
                                     QJsonObject{{QStringLiteral("scanId"), QStringLiteral("nope")},
                                                 {QStringLiteral("view"), QJsonObject{}}}, spy);
    const QJsonObject err = res.value(QStringLiteral("__error")).toObject();
    QCOMPARE(err.value(QStringLiteral("code")).toString(), QStringLiteral("unknown_scan"));
    QVERIFY(!err.value(QStringLiteral("message")).toString().isEmpty());

    const QJsonObject res2 = callSync(client, QStringLiteral("details"),
                                      QJsonObject{{QStringLiteral("scanId"), QStringLiteral("nope")},
                                                  {QStringLiteral("rowId"), QStringLiteral("x")}}, spy);
    QCOMPARE(res2.value(QStringLiteral("__error")).toObject()
                 .value(QStringLiteral("code")).toString(),
             QStringLiteral("unknown_scan"));

    client.stop();
}

QTEST_GUILESS_MAIN(ActionsTest)
#include "actions_test.moc"
