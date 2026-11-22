#pragma once

#include <QColor>
#include <QHash>
#include <QPalette>

// PaletteUtil resolves colors exclusively from QPalette roles so the app
// follows the global Qt theme. Derived shades are produced by blending two
// palette roles; nothing here hardcodes a color.
class PaletteUtil
{
public:
    // Returns the color for a palette role, cached until invalidate().
    static QColor color(QPalette::ColorRole role);

    // Linear blend of two palette roles: t=0 -> a, t=1 -> b.
    static QColor blend(QPalette::ColorRole a, QPalette::ColorRole b, qreal t);

    // Drops the cache; call on ApplicationPaletteChange / PaletteChange.
    static void invalidate();

private:
    static void rebuild();

    static QHash<int, QColor> s_cache;
    static bool s_cached;
};
