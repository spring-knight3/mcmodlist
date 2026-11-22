#pragma once

#include <QJsonObject>
#include <QObject>
#include <QString>

class EngineClient;

// ViewOptions mirrors the engine "view" params.
struct ViewOptions {
    QString search;
    QString loader;
    QString side;
    QString author;
    QString sort;     // "", "field", or "-field"
    QString groupBy;  // "", "loader", or "side"
    bool showDisabled = false;
    bool showNested = false;
    bool unrecognisedOnly = false;
};

// ViewController issues view requests for a scanned instance. Latest-wins:
// responses older than the newest request are dropped.
class ViewController : public QObject
{
    Q_OBJECT
public:
    explicit ViewController(EngineClient *engine, QObject *parent = nullptr);

    void setScanId(const QString &scanId);
    QString scanId() const { return m_scanId; }

    void setOptions(const ViewOptions &options);
    const ViewOptions &options() const { return m_options; }

    // Sends a view request for the current options (latest-wins).
    void refresh();

signals:
    void stateReady(const QJsonObject &state);
    void viewError(const QString &code, const QString &message);

private slots:
    void onResponse(quint64 id, bool ok, const QJsonObject &result, const QJsonObject &error);

private:
    EngineClient *m_engine;
    QString m_scanId;
    ViewOptions m_options;
    quint64 m_seq = 0;
    quint64 m_pendingId = 0;
};
