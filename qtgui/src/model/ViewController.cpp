#include "ViewController.h"

#include "../engine/EngineClient.h"

ViewController::ViewController(EngineClient *engine, QObject *parent)
    : QObject(parent)
    , m_engine(engine)
{
    connect(m_engine, &EngineClient::response, this, &ViewController::onResponse);
}

void ViewController::setScanId(const QString &scanId)
{
    m_scanId = scanId;
}

void ViewController::setOptions(const ViewOptions &options)
{
    m_options = options;
    refresh();
}

void ViewController::refresh()
{
    if (m_scanId.isEmpty() || m_engine == nullptr)
        return;
    ++m_seq;
    const quint64 id = m_engine->call(QStringLiteral("view"), QJsonObject{
        {QStringLiteral("scanId"), m_scanId},
        {QStringLiteral("view"), QJsonObject{
             {QStringLiteral("search"), m_options.search},
             {QStringLiteral("loader"), m_options.loader},
             {QStringLiteral("side"), m_options.side},
             {QStringLiteral("author"), m_options.author},
             {QStringLiteral("sort"), m_options.sort},
             {QStringLiteral("groupBy"), m_options.groupBy},
             {QStringLiteral("showDisabled"), m_options.showDisabled},
             {QStringLiteral("showNested"), m_options.showNested},
             {QStringLiteral("unrecognisedOnly"), m_options.unrecognisedOnly},
         }}});
    // Latest-wins bookkeeping: a newer refresh supersedes this one.
    if (m_pendingId != 0 && m_pendingId != id) {
        // The engine has no view cancellation; stale responses are dropped
        // by seq in onResponse.
    }
    m_pendingId = id;
}

void ViewController::onResponse(quint64 id, bool ok, const QJsonObject &result, const QJsonObject &error)
{
    if (id != m_pendingId)
        return; // stale response from a superseded view request
    if (!ok) {
        emit viewError(error.value(QStringLiteral("code")).toString(),
                       error.value(QStringLiteral("message")).toString());
        return;
    }
    emit stateReady(result.value(QStringLiteral("state")).toObject());
}
