#include <QFile>
#include <QJsonObject>
#include <QSignalSpy>
#include <QTemporaryDir>
#include <QTest>
#include <QTimer>

#include "engine/EngineClient.h"
#include "engine/EngineLocator.h"

class EngineClientTest : public QObject
{
    Q_OBJECT
private slots:
    void stubHelloAndUnknown();
    void splitChunksAssembled();
    void crashMidRequest();
    void protocolMismatch();
    void heartbeatDuringSlowCall();
    void oversizedLineParsed();
    void realEngine();
};

static QString stubPath(const QString &name)
{
    return QStringLiteral(MCMODLIST_QT_TESTS_DIR) + QLatin1Char('/') + name;
}

void EngineClientTest::stubHelloAndUnknown()
{
    EngineClient client;
    client.setEnginePath(stubPath(QStringLiteral("stub-engine.sh")));
    QVERIFY(client.start());

    QSignalSpy spy(&client, &EngineClient::response);
    client.callHello();
    QTRY_VERIFY_WITH_TIMEOUT(spy.count() >= 1, 5000);
    const auto args = spy.takeFirst();
    QCOMPARE(args.at(0).toULongLong(), quint64(1));
    QVERIFY(args.at(1).toBool());
    QCOMPARE(args.at(2).toJsonObject().value(QStringLiteral("protocol")).toInt(), 1);

    client.call(QStringLiteral("bogus"));
    QTRY_VERIFY_WITH_TIMEOUT(spy.count() >= 1, 5000);
    const auto errArgs = spy.takeFirst();
    QVERIFY(!errArgs.at(1).toBool());
    QCOMPARE(errArgs.at(3).toJsonObject().value(QStringLiteral("code")).toString(),
             QStringLiteral("invalid_request"));
    client.stop();
}

void EngineClientTest::splitChunksAssembled()
{
    EngineClient client;
    client.setEnginePath(stubPath(QStringLiteral("stub-engine.sh")));
    QVERIFY(client.start());

    QSignalSpy spy(&client, &EngineClient::response);
    client.call(QStringLiteral("split"));
    QTRY_VERIFY_WITH_TIMEOUT(spy.count() >= 1, 5000);
    const auto args = spy.takeFirst();
    QVERIFY(args.at(1).toBool());
    QVERIFY(args.at(2).toJsonObject().contains(QStringLiteral("split")));
    client.stop();
}

void EngineClientTest::crashMidRequest()
{
    EngineClient client;
    client.setEnginePath(stubPath(QStringLiteral("stub-engine.sh")));
    QVERIFY(client.start());

    QSignalSpy exited(&client, &EngineClient::engineExited);
    QSignalSpy spy(&client, &EngineClient::response);
    const quint64 id = client.call(QStringLiteral("die"));
    QTRY_VERIFY_WITH_TIMEOUT(exited.count() >= 1, 5000);
    QTest::qWait(300);
    bool sawBye = false;
    for (const auto &item : spy) {
        if (item.at(0).toULongLong() == id && item.at(1).toBool())
            sawBye = true;
    }
    QVERIFY(sawBye || spy.count() >= 1);
    client.stop();
}

void EngineClientTest::protocolMismatch()
{
    EngineClient client;
    client.setEnginePath(stubPath(QStringLiteral("mismatch-engine.sh")));
    QVERIFY(client.start());

    QSignalSpy spy(&client, &EngineClient::protocolMismatch);
    client.callHello();
    QTRY_VERIFY_WITH_TIMEOUT(spy.count() >= 1, 5000);
    QVERIFY(spy.takeFirst().at(0).toString().contains(QStringLiteral("mismatch")));
    client.stop();
}

void EngineClientTest::heartbeatDuringSlowCall()
{
    EngineClient client;
    client.setEnginePath(stubPath(QStringLiteral("stub-engine.sh")));
    QVERIFY(client.start());

    int ticks = 0;
    QTimer timer;
    timer.setInterval(50);
    QObject::connect(&timer, &QTimer::timeout, [&]() { ++ticks; });
    timer.start();

    QSignalSpy spy(&client, &EngineClient::response);
    client.call(QStringLiteral("slow"));
    QTRY_VERIFY_WITH_TIMEOUT(spy.count() >= 1, 5000);
    QVERIFY(ticks >= 5);
    client.stop();
}

void EngineClientTest::oversizedLineParsed()
{
    EngineClient client;
    client.setEnginePath(stubPath(QStringLiteral("stub-engine.sh")));
    QVERIFY(client.start());

    QSignalSpy spy(&client, &EngineClient::response);
    client.call(QStringLiteral("big"));
    QTRY_VERIFY_WITH_TIMEOUT(spy.count() >= 1, 15000);
    const auto args = spy.takeFirst();
    QVERIFY(args.at(1).toBool());
    QVERIFY(args.at(2).toJsonObject().value(QStringLiteral("blob")).toString().size() > 200000);
    client.stop();
}

void EngineClientTest::realEngine()
{
    const QString path = findEnginePath();
    if (path.isEmpty())
        QSKIP("mcmodlist-engine not found; build it or set MCMODLIST_ENGINE");

    EngineClient client;
    client.setEnginePath(path);
    QVERIFY(client.start());

    QSignalSpy spy(&client, &EngineClient::response);
    client.callHello();
    QTRY_VERIFY_WITH_TIMEOUT(spy.count() >= 1, 10000);
    const auto args = spy.takeFirst();
    QVERIFY(args.at(1).toBool());
    QCOMPARE(args.at(2).toJsonObject().value(QStringLiteral("protocol")).toInt(), 1);

    QSignalSpy listSpy(&client, &EngineClient::response);
    client.call(QStringLiteral("listInstances"));
    QTRY_VERIFY_WITH_TIMEOUT(listSpy.count() >= 1, 10000);
    client.stop();
}

QTEST_GUILESS_MAIN(EngineClientTest)
#include "engineclient_test.moc"
