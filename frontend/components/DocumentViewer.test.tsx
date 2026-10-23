import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import DocumentViewer from "./DocumentViewer";

describe("DocumentViewer", () => {
  it("renders markdown content with initial markdown", () => {
    const { container } = render(
      <LanguageProvider>
        <DocumentViewer
          documentKey="resources/documents/doc001/document.md"
          documentMimeType="text/markdown"
          initialMarkdown={`# Sample Title

Sample body text.`}
        />
      </LanguageProvider>,
    );

    expect(container).toBeDefined();
    expect(screen.getByText("Sample Title")).toBeDefined();
    expect(screen.getByText("Sample body text.")).toBeDefined();
  });

  it("renders iframe for PDF document", () => {
    const { container } = render(
      <LanguageProvider>
        <DocumentViewer
          documentKey="resources/documents/doc002/document.pdf"
          documentMimeType="application/pdf"
        />
      </LanguageProvider>,
    );

    expect(container).toBeDefined();
    const iframe = container.querySelector("iframe");
    expect(iframe).not.toBeNull();
    expect(iframe?.getAttribute("src")).toContain("#toolbar=0&navpanes=0");
  });

  it("renders not found state when key is missing", () => {
    render(
      <LanguageProvider>
        <DocumentViewer documentKey={null} documentMimeType={null} />
      </LanguageProvider>,
    );

    expect(screen.getByText("Document not found")).toBeDefined();
  });
});
