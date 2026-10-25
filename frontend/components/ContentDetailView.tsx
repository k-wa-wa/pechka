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
              <ChannelLink channelId={content.channel_id} channelName={content.channel_name} />
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

          {/* Tags */}
          <TagList tags={content.tags} />
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
            <ChannelLink channelId={content.channel_id} channelName={content.channel_name} />
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

        {/* Tags */}
        <TagList tags={content.tags} />
      </div>
    </div>
  );
}

function ChannelLink({ channelId, channelName }: { channelId: string; channelName?: string }) {
  return (
    <Link
      href={`/channels/${channelId}`}
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: 6,
        padding: "4px 10px",
        borderRadius: "var(--radius-full)",
        backgroundColor: "var(--accent-subtle)",
        border: "1px solid var(--accent-border)",
        color: "var(--accent)",
        fontSize: 13,
        fontWeight: 600,
        textDecoration: "none",
      }}
    >
      <svg
        aria-hidden="true"
        data-testid="channel-icon"
        width="14"
        height="14"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="2" y="7" width="20" height="15" rx="2" />
        <polyline points="17 2 12 7 7 2" />
      </svg>
      <span>{channelName || channelId}</span>
      <span aria-hidden="true">›</span>
    </Link>
  );
}

function TagList({ tags }: { tags: string[] }) {
  if (tags.length === 0) return null;
  return (
    <div
      style={{
        display: "flex",
        flexWrap: "wrap",
        gap: "4px 12px",
        marginTop: 12,
        fontSize: 13,
        color: "var(--muted)",
      }}
    >
      {tags.map((tag) => (
        <span key={tag}>#{tag}</span>
      ))}
    </div>
  );
}
