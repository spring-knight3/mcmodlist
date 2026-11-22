#include <QDir>
#include <QFile>
#include <QJsonDocument>
#include <QJsonObject>
#include <QTest>

// Golden protocol files shared with the Go engine tests.
class ProtocolTest : public QObject
{
    Q_OBJECT
private slots:
    void goldensParseAndVersioned();
};

void ProtocolTest::goldensParseAndVersioned()
{
    const QDir dir(QStringLiteral(MCMODLIST_TESTDATA_DIR));
    QVERIFY(dir.exists());
    const QStringList files = dir.entryList({QStringLiteral("*.json")}, QDir::Files);
    QVERIFY(files.size() >= 10);

    int requests = 0, responses = 0, events = 0;
    for (const QString &name : files) {
        QFile f(dir.filePath(name));
        QVERIFY2(f.open(QIODevice::ReadOnly), qPrintable(name));
        const QJsonDocument doc = QJsonDocument::fromJson(f.readAll());
        f.close();

        if (name == QLatin1String("malformed.request.json")) {
            QVERIFY(!doc.isObject());
            continue;
        }
        QVERIFY2(doc.isObject(), qPrintable(name));
        const QJsonObject obj = doc.object();

        const int wantV = (name == QLatin1String("unsupported_version.request.json")) ? 2 : 1;
        QCOMPARE(obj.value(QStringLiteral("v")).toInt(), wantV);

        if (name.endsWith(QLatin1String(".request.json"))) {
            QVERIFY(obj.contains(QStringLiteral("method")));
            ++requests;
        } else if (name.endsWith(QLatin1String(".response.json"))) {
            QVERIFY(obj.contains(QStringLiteral("result")) || obj.contains(QStringLiteral("error")));
            ++responses;
        } else if (name.endsWith(QLatin1String(".event.json"))) {
            QVERIFY(obj.contains(QStringLiteral("event")));
            ++events;
        }
    }
    QVERIFY(requests >= 12);
    QVERIFY(responses >= 4);
    QVERIFY(events >= 1);
}

QTEST_GUILESS_MAIN(ProtocolTest)
#include "protocol_test.moc"
