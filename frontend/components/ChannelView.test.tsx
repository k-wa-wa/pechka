import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import { CHANNELS, CONTENTS } from "@/mocks/fixtures";
import ChannelView from "./ChannelView";

describe("ChannelView", () => {
  it("renders channel details and contents", () => {
    render(
      <LanguageProvider>
        <ChannelView channel={CHANNELS[0]} contents={CONTENTS} currentType="" />
      </LanguageProvider>,
    );

    expect(screen.getAllByText(CHANNELS[0].name).length).toBeGreaterThan(0);
  });
});
