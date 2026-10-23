import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import VrViewer from "./VrViewer";

describe("VrViewer", () => {
  it("renders vr viewer container", () => {
    const { container } = render(<VrViewer variants={[]} />);

    expect(container).toBeDefined();
  });
});
