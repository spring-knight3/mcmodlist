#pragma once

#include <QHash>
#include <QJsonObject>
#include <QObject>
#include <atomic>
#include <QProcess>
#include <QString>

class QTimer;

// EngineClient speaks the mcmodlist JSON-Lines protocol to a
// mcmodlist-engine child process over stdin/stdout.
//
// The GUI thread is never blocked: all parsing happens in readyRead handlers,
// and lines over kLongLineBytes are parsed in a QtConcurrent worker.
class EngineClient : public QObject
{
    Q_OBJECT
public:
    explicit EngineClient(QObject *parent = nullptr);
    ~EngineClient() override;

    void setEnginePath(const QString &path);
    QString enginePath() const { return m_path; }
    bool isRunning() const;

    // Starts the engine process. Does not send hello; call callHello().
    bool start();
    void stop();

    // Sends a request and returns its id. Responses arrive via the response()
    // signal; scan-style progress arrives via eventReceived().
    quint64 call(const QString &method, const QJsonObject &params = QJsonObject());

    // Sends hello and validates the protocol version in the reply.
    quint64 callHello();

    static constexpr int kProtocolVersion = 1;
    static constexpr int kLongLineBytes = 256 * 1024;
    static constexpr int kRequestTimeoutMs = 30000;

signals:
    void response(quint64 id, bool ok, const QJsonObject &result, const QJsonObject &error);
    void eventReceived(quint64 id, const QString &event, const QJsonObject &data);
    void protocolMismatch(const QString &message);
    void engineError(const QString &message);
    void engineExited(int exitCode, QProcess::ExitStatus status);
    void stderrLine(const QString &line);

private slots:
    void onReadyRead();
    void onStdErr();
    void onFinished(int exitCode, QProcess::ExitStatus status);
    void onTimeout(quint64 id);

private:
    void sendLine(const QJsonObject &obj);
    void handleLine(const QByteArray &line);
    void dispatchParsed(const QByteArray &line);
    void queueFinish(quint64 id);
    void finishRequest(quint64 id);

    QString m_path;
    QProcess *m_proc = nullptr;
    QByteArray m_buf;
    QByteArray m_errBuf;
    quint64 m_nextId = 1;
    std::atomic<quint64> m_helloId{0};
    QHash<quint64, QString> m_pending;
    QHash<quint64, QTimer *> m_timers;
};
