"use client";

import Link from "next/link";
import ContentPlayer from "@/components/ContentPlayer";
import DocumentViewer from "@/components/DocumentViewer";
import { useLanguage } from "@/lib/i18n/LanguageContext";
import type { MongoContent, MongoVariant } from "@/lib/types";

interface Props {
  content: MongoContent;
  variants: MongoVariant[];
}

export default function ContentDetailView({ content, variants }: Props) {
  const { t, language } = useLanguage();
  const isVr = content.content_type === "vr360";
  const isDocument = content.content_type === "document";

  if (isDocument) {
    const resourceUrl = content.document_key
      ? content.document_key.startsWith("/")
        ? content.document_key
        : `/${content.document_key}`
      : null;
    const isPdf =
      content.document_mime_type === "application/pdf" ||
      content.document_key?.toLowerCase().endsWith(".pdf");
    const formatBadge = isPdf ? "PDF" : "Markdown";

    return (
      <div
        style={{
          maxWidth: 960,
          margin: "0 auto",
          padding: "32px 24px",
        }}
      >
        {/* Document Header Section */}
        <header style={{ marginBottom: 24 }}>
          {/* Tags */}
          {content.tags.length > 0 && (
            <div
              style={{
                display: "flex",
                flexWrap: "wrap",
                gap: 6,
                marginBottom: 12,
              }}
            >
              {content.tags.map((tag) => (
                <span
                  key={tag}
                  style={{
                    fontSize: 12,
                    padding: "3px 8px",
                    borderRadius: "var(--radius-sm)",
                    backgroundColor: "var(--accent-subtle)",
                    color: "var(--accent)",
                    border: "1px solid var(--accent-border)",
                  }}
                >
                  {tag}
                </span>
              ))}
            </div>
          )}

          {/* Title */}
          <h1
            style={{
              margin: "0 0 16px",
              fontSize: "clamp(22px, 4vw, 32px)",
              fontWeight: 700,
              color: "var(--fg)",
              lineHeight: 1.3,
            }}
          >
            {content.title}
          </h1>

          {/* Meta & Actions bar */}
          <div
            style={{
              display: "flex",
              alignItems: "center",
              justifyContent: "space-between",
              gap: 16,
              flexWrap: "wrap",
              paddingBottom: 16,
              borderBottom: "1px solid var(--line)",
              fontSize: 13,
              color: "var(--muted)",
            }}
          >
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: 12,
                flexWrap: "wrap",
              }}
            >
              <span
                style={{
                  fontSize: 11,
                  fontWeight: 600,
                  padding: "2px 8px",
                  borderRadius: "var(--radius-sm)",
                  backgroundColor: "var(--accent-emphasis)",
                  color: "var(--accent)",
                  border: "1px solid var(--accent-border)",
                }}
              >
                {formatBadge}
              </span>
              {content.channel_id && (
                <Link
                  href={`/channels/${content.channel_id}`}
                  style={{
                    display: "inline-flex",
                    alignItems: "center",
                    gap: 6,
                    padding: "3px 10px",
                    borderRadius: "var(--radius-full)",
                    backgroundColor: "var(--card)",
                    border: "1px solid var(--line)",
                    color: "var(--fg)",
                    fontSize: 12,
                    fontWeight: 600,
                    textDecoration: "none",
                  }}
                >
                  <span>{content.channel_name || content.channel_id}</span>
                </Link>
              )}
              {content.published_at && (
                <span>
                  {new Date(content.published_at).toLocaleDateString(
                    language === "ja" ? "ja-JP" : "en-US",
                    { timeZone: "Asia/Tokyo" },
                  )}{" "}
                  {t("content.published")}
                </span>
              )}
            </div>

            {/* Action buttons (Open new tab, Download) */}
            {resourceUrl && (
              <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                <a
                  href={resourceUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  style={{
                    padding: "5px 12px",
                    borderRadius: "var(--radius-sm)",
                    border: "1px solid var(--line)",
                    backgroundColor: "var(--card)",
                    color: "var(--fg)",
                    fontSize: 12,
                    textDecoration: "none",
                    display: "inline-flex",
                    alignItems: "center",
                    gap: 4,
                  }}
                >
                  {t("document.openNewTab")}
                </a>
                <a
                  href={resourceUrl}
                  download
                  style={{
                    padding: "5px 12px",
                    borderRadius: "var(--radius-sm)",
                    border: "1px solid var(--accent-border)",
                    backgroundColor: "var(--accent-subtle)",
                    color: "var(--accent)",
                    fontSize: 12,
                    textDecoration: "none",
                    display: "inline-flex",
                    alignItems: "center",
                    gap: 4,
                  }}
                >
                  {t("document.download")}
                </a>
              </div>
            )}
          </div>

          {/* Description */}
          {content.description && (
            <p
              style={{
                margin: "16px 0 0",
                fontSize: 14,
                color: "var(--muted)",
                lineHeight: 1.6,
                whiteSpace: "pre-wrap",
              }}
            >
              {content.description}
            </p>
          )}
        </header>

        {/* Document Viewer Body */}
        <DocumentViewer
          documentKey={content.document_key}
          documentMimeType={content.document_mime_type}
        />
      </div>
    );
  }

  return (
    <div
      style={{
        maxWidth: isVr ? 1280 : 960,
        margin: "0 auto",
        padding: isVr ? "0 0 32px" : "32px 24px",
      }}
    >
      <ContentPlayer
        variants={variants}
        isVr={isVr}
        shortId={content.short_id}
        hasSubtitles={content.has_subtitles}
      />
      <div
        style={{
          maxWidth: isVr ? 900 : undefined,
          margin: isVr ? "24px auto 0" : "24px 0 0",
          padding: isVr ? "0 24px" : undefined,
        }}
      >
        {/* Tags */}
        {content.tags.length > 0 && (
          <div
            style={{
              display: "flex",
              flexWrap: "wrap",
              gap: 6,
              marginBottom: 16,
            }}
          >
            {content.tags.map((tag) => (
              <span
                key={tag}
                style={{
                  fontSize: 12,
                  padding: "3px 8px",
                  borderRadius: "var(--radius-sm)",
                  backgroundColor: "var(--accent-subtle)",
                  color: "var(--accent)",
                  border: "1px solid var(--accent-border)",
                }}
              >
                {tag}
              </span>
            ))}
          </div>
        )}

        {/* Title */}
        <h1
          style={{
            margin: "0 0 12px",
            fontSize: "clamp(20px, 4vw, 28px)",
            fontWeight: 700,
            color: "var(--fg)",
            lineHeight: 1.3,
          }}
        >
          {content.title}
        </h1>

        {/* Meta */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 16,
            marginBottom: 16,
            flexWrap: "wrap",
            fontSize: 13,
            color: "var(--muted)",
          }}
        >
          {content.channel_id && (
            <Link
              href={`/channels/${content.channel_id}`}
              style={{
                display: "inline-flex",
                alignItems: "center",
                gap: 6,
                padding: "3px 10px",
                borderRadius: "var(--radius-full)",
                backgroundColor: "var(--card)",
                border: "1px solid var(--line)",
                color: "var(--fg)",
                fontSize: 12,
                fontWeight: 600,
                textDecoration: "none",
              }}
            >
              <span>{content.channel_name || content.channel_id}</span>
            </Link>
          )}
          {content.duration_seconds != null && (
            <span>
              {Math.floor(content.duration_seconds / 60)}
              {t("content.minutes")}
              {content.duration_seconds % 60}
              {t("content.seconds")}
            </span>
          )}
          {content.published_at && (
            <span>
              {new Date(content.published_at).toLocaleDateString(
                language === "ja" ? "ja-JP" : "en-US",
                { timeZone: "Asia/Tokyo" },
              )}{" "}
              {t("content.published")}
            </span>
          )}
        </div>

        {/* Description */}
        {content.description && (
          <p
            style={{
              margin: 0,
              fontSize: 15,
              color: "var(--muted)",
              lineHeight: 1.7,
              whiteSpace: "pre-wrap",
            }}
          >
            {content.description}
          </p>
        )}
      </div>
    </div>
  );
}
