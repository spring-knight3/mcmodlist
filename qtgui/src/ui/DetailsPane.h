#pragma once

#include <QJsonObject>
#include <QWidget>

class QLabel;
class QVBoxLayout;

// DetailsPane shows engine-provided details for the selected mod:
// description, authors, license, links, jar path, and warnings.
class DetailsPane : public QWidget
{
    Q_OBJECT
public:
    explicit DetailsPane(QWidget *parent = nullptr);

    void setDetails(const QJsonObject &details);
    void clearDetails();

protected:
    bool event(QEvent *event) override;

private:
    void addField(const QString &label, const QString &value);
    void applyWarningPalette();

    QVBoxLayout *m_layout;
    QLabel *m_title;
    QLabel *m_description;
    QWidget *m_fields;
    QVBoxLayout *m_fieldsLayout;
    QLabel *m_warnings;
    QString m_warningsText;
    bool m_hasWarnings = false;
};
