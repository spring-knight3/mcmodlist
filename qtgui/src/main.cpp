#include <QApplication>
#include <QCoreApplication>
#include <QSettings>

#include "ui/MainWindow.h"

int main(int argc, char *argv[])
{
    QApplication app(argc, argv);
    QCoreApplication::setApplicationName(QStringLiteral("mcmodlist"));
    QCoreApplication::setApplicationVersion(QStringLiteral("dev"));
    QCoreApplication::setOrganizationName(QStringLiteral("mcmodlist"));

    MainWindow window;
    window.show();
    return app.exec();
}
