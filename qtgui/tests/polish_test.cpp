#include <QAbstractButton>
#include <QApplication>
#include <QDir>
#include <QImage>
#include <QJsonArray>
#include <QJsonObject>
#include <QLabel>
#include <QLineEdit>
#include <QListWidget>
#include <QPainter>
#include <QSettings>
#include <QStyleOptionViewItem>
#include <QTableView>
#include <QTemporaryDir>
#include <QTest>

#include "engine/EngineLocator.h"
#include "model/ModTableModel.h"
#include "ui/ChipBar.h"
#include "ui/DetailsPane.h"
#include "ui/MainWindow.h"
#include "ui/ModTableView.h"
#include "ui/PaletteUtil.h"
#include "ui/StartScreen.h"

class PolishTest : public QObject
{
    Q_OBJECT
private slots:
    void initTestCase();
    void groupHeaderFollowsPalette();
    void detailsWarningsFollowPalette();
    void settingsRoundTrip();
    void accessibleNamesAndFocus();
    void cleanupTestCase();

private:
    QTemporaryDir m_configDir;
};

void PolishTest::initTestCase()
{
    // Isolate QSettings so tests never touch the real user config.
    QVERIFY(m_configDir.isValid());
    qputenv("XDG_CONFIG_HOME", m_configDir.path().toUtf8());
    QCoreApplication::setOrganizationName(QStringLiteral("mcmodlist-test"));
    QCoreApplication::setApplicationName(QStringLiteral("polish-test"));
}

static QJsonObject row(const QString &name)
{
    return QJsonObject{{QStringLiteral("rowId"), name},
                       {QStringLiteral("name"), name},
                       {QStringLiteral("id"), name},
                       {QStringLiteral("version"), QStringLiteral("1.0")}};
}

void PolishTest::groupHeaderFollowsPalette()
{
    ModTableModel model;
    model.setState(QJsonObject{
        {QStringLiteral("instance"), QJsonObject{{QStringLiteral("name"), QStringLiteral("T")}}},
        {QStringLiteral("total"), 2},
        {QStringLiteral("rows"), QJsonArray{row(QStringLiteral("a")), row(QStringLiteral("b"))}},
        {QStringLiteral("groups"),
         QJsonArray{QJsonObject{{QStringLiteral("start"), 0},
                                {QStringLiteral("count"), 2},
                                {QStringLiteral("title"), QStringLiteral("Fabric")}}}},
    });
    QVERIFY(model.isGroupHeader(0));

    ModTableView view;
    view.setModel(&model);
    view.resize(400, 200);
    view.show();

    auto paintHeader = [&view, &model]() {
        QImage img(240, 28, QImage::Format_ARGB32);
        img.fill(Qt::transparent);
        QPainter painter(&img);
        QStyleOptionViewItem opt;
        opt.rect = QRect(0, 0, 240, 28);
        opt.widget = &view;
        opt.state = QStyle::State_Enabled;
        view.itemDelegate()->paint(&painter, opt, model.index(0, 0));
        painter.end();
        // Sample the bottom-right corner: guaranteed background, away from text.
        return img.pixelColor(img.width() - 2, img.height() - 2);
    };

    PaletteUtil::invalidate();
    const QColor before = paintHeader();

    QPalette pal = qApp->palette();
    pal.setColor(QPalette::AlternateBase, QColor(Qt::blue));
    pal.setColor(QPalette::WindowText, QColor(Qt::red));
    qApp->setPalette(pal);
    PaletteUtil::invalidate();
    view.viewport()->update();

    const QColor after = paintHeader();
    QVERIFY(before != after);
    QCOMPARE(PaletteUtil::color(QPalette::AlternateBase), QColor(Qt::blue));

    qApp->setPalette(QApplication::style()->standardPalette());
    PaletteUtil::invalidate();
}

void PolishTest::detailsWarningsFollowPalette()
{
    DetailsPane pane;
    pane.setDetails(QJsonObject{
        {QStringLiteral("mod"),
         QJsonObject{{QStringLiteral("name"), QStringLiteral("M")},
                     {QStringLiteral("id"), QStringLiteral("m")}}},
        {QStringLiteral("warnings"),
         QJsonArray{QJsonObject{{QStringLiteral("type"), QStringLiteral("unreadable")},
                                {QStringLiteral("message"), QStringLiteral("bad jar")}}}},
    });
    pane.show();

    QLabel *warnings = nullptr;
    for (QLabel *label : pane.findChildren<QLabel *>()) {
        if (label->text().contains(QStringLiteral("⚠"))) {
            warnings = label;
            break;
        }
    }
    QVERIFY(warnings);
    QCOMPARE(warnings->palette().color(QPalette::Text), PaletteUtil::color(QPalette::BrightText));

    QPalette pal = qApp->palette();
    pal.setColor(QPalette::BrightText, QColor(Qt::green));
    qApp->setPalette(pal);
    PaletteUtil::invalidate();
    QEvent ev(QEvent::ApplicationPaletteChange);
    QCoreApplication::sendEvent(&pane, &ev);

    QCOMPARE(warnings->palette().color(QPalette::Text), PaletteUtil::color(QPalette::BrightText));
    QVERIFY(warnings->palette().color(QPalette::Text) == QColor(Qt::green));

    qApp->setPalette(QApplication::style()->standardPalette());
    PaletteUtil::invalidate();
}

void PolishTest::settingsRoundTrip()
{
    if (findEnginePath().isEmpty())
        QSKIP("mcmodlist-engine not found");

    const QString envEngine = qEnvironmentVariable("MCMODLIST_ENGINE");
    QVERIFY(!envEngine.isEmpty());

    auto *w1 = new MainWindow;
    w1->resize(777, 555);
    w1->show();
    QTest::qWait(100);
    const QSize savedSize = w1->size();
    QVERIFY(savedSize.isValid());
    w1->close();
    delete w1;

    {
        QSettings settings;
        QVERIFY(!settings.value(QStringLiteral("geometry")).toByteArray().isEmpty());
        QVERIFY(!settings.value(QStringLiteral("splitter")).toByteArray().isEmpty());
        // An explicit env engine override is persisted.
        QCOMPARE(settings.value(QStringLiteral("enginePath")).toString(), envEngine);
    }

    auto *w2 = new MainWindow;
    w2->show();
    QTest::qWait(50);
    QCOMPARE(w2->size(), savedSize);
    w2->close();
    delete w2;
}

void PolishTest::accessibleNamesAndFocus()
{
    StartScreen start;
    start.show();
    for (QAbstractButton *btn : start.findChildren<QAbstractButton *>())
        QVERIFY2(!btn->accessibleName().isEmpty(), qPrintable(btn->text()));
    QVERIFY(!start.findChild<QLineEdit *>()->accessibleName().isEmpty());
    QVERIFY(start.findChild<QListWidget *>() != nullptr);

    ChipBar chips;
    chips.setOptions({QStringLiteral("fabric")});
    chips.show();
    for (QAbstractButton *btn : chips.findChildren<QAbstractButton *>())
        QVERIFY2(!btn->accessibleName().isEmpty(), qPrintable(btn->text()));

    // Every button remains keyboard-focusable (no NoFocus policies).
    for (QWidget *w : start.findChildren<QWidget *>()) {
        if (auto *btn = qobject_cast<QAbstractButton *>(w))
            QVERIFY(btn->focusPolicy() != Qt::NoFocus);
    }
}

void PolishTest::cleanupTestCase()
{
    qApp->setPalette(QApplication::style()->standardPalette());
    PaletteUtil::invalidate();
}

QTEST_MAIN(PolishTest)
#include "polish_test.moc"
