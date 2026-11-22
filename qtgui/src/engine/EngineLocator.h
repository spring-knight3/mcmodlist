#pragma once

#include <QString>

// Locates the mcmodlist-engine binary. Search order:
// 1. MCMODLIST_ENGINE environment variable
// 2. "enginePath" in QSettings (saved override)
// 3. mcmodlist-engine next to this executable
// 4. PATH
QString findEnginePath();
