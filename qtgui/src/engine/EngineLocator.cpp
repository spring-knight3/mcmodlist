#include "EngineLocator.h"

#include <QCoreApplication>
#include <QDir>
#include <QFileInfo>
#include <QSettings>
#include <QStandardPaths>

namespace {

bool usable(const QString &path)
{
    return !path.isEmpty() && QFileInfo::exists(path) && QFileInfo(path).isExecutable();
}

} // namespace

QString findEnginePath()
{
    const QString fromEnv = qEnvironmentVariable("MCMODLIST_ENGINE");
    if (usable(fromEnv))
        return fromEnv;

    const QString fromSettings = QSettings().value(QStringLiteral("enginePath")).toString();
    if (usable(fromSettings))
        return fromSettings;

    const QString beside = QDir(QCoreApplication::applicationDirPath())
                               .filePath(QStringLiteral("mcmodlist-engine"));
    if (usable(beside))
        return beside;

    return QStandardPaths::findExecutable(QStringLiteral("mcmodlist-engine"));
}
