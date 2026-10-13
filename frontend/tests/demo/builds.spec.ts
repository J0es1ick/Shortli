import { readFileSync, readdirSync } from "node:fs";
import { expect, test } from "@playwright/test";

const javascript = (directory: string) =>
  readdirSync(directory)
    .filter((file) => file.endsWith(".js"))
    .map((file) => readFileSync(`${directory}/${file}`, "utf8"))
    .join("\n");

test("demo storage, synthetic keys and role bypass are excluded from the server build", () => {
  const production = javascript("dist/assets");
  const demo = javascript("dist-demo/assets");
  for (const marker of [
    "shortli:demo:v1",
    "demo_not_a_real_key_",
    "/api/demo/role",
    "/api/demo/reset",
  ]) {
    expect(production).not.toContain(marker);
    expect(demo).toContain(marker);
  }
});
