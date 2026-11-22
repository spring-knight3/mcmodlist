#include "EngineClient.h"

#include <QCoreApplication>
#include <QJsonDocument>
#include <QJsonValue>
#include <QTimer>
#include <QtConcurrentRun>

EngineClient::EngineClient(QObject *parent)
    : QObject(parent)
{
}

EngineClient::~EngineClient()
{
    stop();
}

void EngineClient::setEnginePath(const QString &path)
{
    m_path = path;
}

bool EngineClient::isRunning() const
{
    return m_proc && m_proc->state() != QProcess::NotRunning;
}

bool EngineClient::start()
{
    if (m_path.isEmpty()) {
        emit engineError(QStringLiteral("engine path is empty"));
        return false;
    }
    if (isRunning())
        return true;

    m_proc = new QProcess(this);
    connect(m_proc, &QProcess::readyRead, this, &EngineClient::onReadyRead);
    connect(m_proc, &QProcess::readyReadStandardError, this, &EngineClient::onStdErr);
    connect(m_proc, &QProcess::finished, this, &EngineClient::onFinished);
    m_proc->start(m_path);
    if (!m_proc->waitForStarted(5000)) {
        emit engineError(QStringLiteral("failed to start engine: %1").arg(m_path));
        return false;
    }
    m_buf.clear();
    return true;
}

void EngineClient::stop()
{
    for (QTimer *timer : m_timers)
        delete timer;
    m_timers.clear();
    m_pending.clear();
    if (m_proc) {
        // Deliberate stop must not emit finished/engineExited: callers may be
        // restarting or destroying the client, and onFinished would touch
        // widgets that may already be gone.
        m_proc->blockSignals(true);
        m_proc->kill();
        m_proc->waitForFinished(2000);
        m_proc->deleteLater();
        m_proc = nullptr;
    }
}

quint64 EngineClient::call(const QString &method, const QJsonObject &params)
{
    const quint64 id = m_nextId++;
    m_pending.insert(id, method);

    auto *timer = new QTimer(this);
    timer->setSingleShot(true);
    timer->setInterval(kRequestTimeoutMs);
    connect(timer, &QTimer::timeout, this, [this, id]() { onTimeout(id); });
    timer->start();
    m_timers.insert(id, timer);

    QJsonObject req;
    req.insert(QStringLiteral("v"), kProtocolVersion);
    req.insert(QStringLiteral("id"), static_cast<double>(id));
    req.insert(QStringLiteral("method"), method);
    if (!params.isEmpty())
        req.insert(QStringLiteral("params"), params);
    sendLine(req);
    return id;
}

quint64 EngineClient::callHello()
{
    QJsonObject params;
    params.insert(QStringLiteral("client"), QStringLiteral("mcmodlist-gui"));
    params.insert(QStringLiteral("clientVersion"), QCoreApplication::applicationVersion());
    const quint64 id = call(QStringLiteral("hello"), params);
    m_helloId.store(id);
    return id;
}

void EngineClient::onTimeout(quint64 id)
{
    if (!m_pending.contains(id))
        return;
    m_pending.remove(id);
    QJsonObject err;
    err.insert(QStringLiteral("code"), QStringLiteral("internal"));
    err.insert(QStringLiteral("message"), QStringLiteral("engine request timed out"));
    emit response(id, false, QJsonObject(), err);
    finishRequest(id);
}

void EngineClient::sendLine(const QJsonObject &obj)
{
    if (!isRunning())
        return;
    QByteArray line = QJsonDocument(obj).toJson(QJsonDocument::Compact);
    line.append('\n');
    m_proc->write(line);
}

void EngineClient::onReadyRead()
{
    m_buf += m_proc->readAllStandardOutput();
    int pos;
    while ((pos = m_buf.indexOf('\n')) >= 0) {
        QByteArray line = m_buf.left(pos);
        m_buf.remove(0, pos + 1);
        if (!line.isEmpty())
            handleLine(line);
    }
}

void EngineClient::handleLine(const QByteArray &line)
{
    if (line.size() > kLongLineBytes) {
        // Parse oversized lines off the GUI thread.
        [[maybe_unused]] auto future = QtConcurrent::run([this, line]() { dispatchParsed(line); });
        return;
    }
    dispatchParsed(line);
}

void EngineClient::dispatchParsed(const QByteArray &line)
{
    QJsonParseError perr;
    const QJsonDocument doc = QJsonDocument::fromJson(line, &perr);
    if (perr.error != QJsonParseError::NoError || !doc.isObject()) {
        emit engineError(QStringLiteral("malformed engine message: %1")
                             .arg(QString::fromUtf8(line.left(200))));
        return;
    }
    const QJsonObject msg = doc.object();
    const double idv = msg.value(QStringLiteral("id")).toDouble(-1);
    const quint64 id = static_cast<quint64>(idv);

    if (msg.contains(QStringLiteral("event"))) {
        const QJsonObject data = msg.value(QStringLiteral("data")).toObject();
        emit eventReceived(id, msg.value(QStringLiteral("event")).toString(), data);
        return;
    }

    if (msg.contains(QStringLiteral("error"))) {
        emit response(id, false, QJsonObject(), msg.value(QStringLiteral("error")).toObject());
        queueFinish(id);
        return;
    }

    const QJsonObject result = msg.value(QStringLiteral("result")).toObject();
    if (id == m_helloId.load() && result.value(QStringLiteral("protocol")).toInt(-1) != kProtocolVersion) {
        emit protocolMismatch(QStringLiteral("engine protocol version mismatch: engine reported %1, client expects %2")
                                  .arg(result.value(QStringLiteral("protocol")).toInt(-1))
                                  .arg(kProtocolVersion));
    }
    emit response(id, true, result, QJsonObject());
    queueFinish(id);
}

void EngineClient::queueFinish(quint64 id)
{
    // dispatchParsed may run on a worker thread for oversized lines; all
    // pending/timer state is GUI-thread only, so finish there.
    QMetaObject::invokeMethod(this, [this, id]() { finishRequest(id); }, Qt::QueuedConnection);
}

void EngineClient::finishRequest(quint64 id)
{
    m_pending.remove(id);
    if (QTimer *timer = m_timers.take(id)) {
        timer->stop();
        timer->deleteLater();
    }
}

void EngineClient::onStdErr()
{
    m_errBuf += m_proc->readAllStandardError();
    int pos;
    while ((pos = m_errBuf.indexOf('\n')) >= 0) {
        const QString line = QString::fromUtf8(m_errBuf.left(pos));
        m_errBuf.remove(0, pos + 1);
        if (!line.isEmpty())
            emit stderrLine(line);
    }
}

void EngineClient::onFinished(int exitCode, QProcess::ExitStatus status)
{
    m_buf.clear();
    // Fail all pending requests so callers are not left hanging.
    const auto pending = m_pending.keys();
    for (quint64 id : pending) {
        QJsonObject err;
        err.insert(QStringLiteral("code"), QStringLiteral("internal"));
        err.insert(QStringLiteral("message"), QStringLiteral("engine exited"));
        emit response(id, false, QJsonObject(), err);
    }
    m_pending.clear();
    for (QTimer *timer : m_timers)
        delete timer;
    m_timers.clear();
    emit engineExited(exitCode, status);
}
