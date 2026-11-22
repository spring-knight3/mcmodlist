#include "PaletteUtil.h"

#include <QApplication>
#include <QPalette>

QHash<int, QColor> PaletteUtil::s_cache;
bool PaletteUtil::s_cached = false;

void PaletteUtil::invalidate()
{
    s_cached = false;
    s_cache.clear();
}

void PaletteUtil::rebuild()
{
    s_cache.clear();
    const QPalette pal = qApp->palette();
    for (int role = QPalette::WindowText; role <= QPalette::NoRole; ++role) {
        const auto cr = static_cast<QPalette::ColorRole>(role);
        s_cache.insert(role, pal.color(cr));
    }
    s_cached = true;
}

QColor PaletteUtil::color(QPalette::ColorRole role)
{
    if (!s_cached)
        rebuild();
    return s_cache.value(static_cast<int>(role), qApp->palette().color(role));
}

QColor PaletteUtil::blend(QPalette::ColorRole a, QPalette::ColorRole b, qreal t)
{
    const QColor ca = color(a);
    const QColor cb = color(b);
    return QColor::fromRgbF(
        ca.redF() + (cb.redF() - ca.redF()) * t,
        ca.greenF() + (cb.greenF() - ca.greenF()) * t,
        ca.blueF() + (cb.blueF() - ca.blueF()) * t);
}
