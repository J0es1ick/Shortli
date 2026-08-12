import LegalPage, { type LegalSection } from "./legalPage";

const sections: LegalSection[] = [
  { title: "terms.acceptance.title", paragraphs: ["terms.acceptance.body"] },
  { title: "terms.accounts.title", paragraphs: ["terms.accounts.body"] },
  {
    title: "terms.links.title",
    paragraphs: ["terms.links.body", "terms.links.immutable"],
  },
  {
    title: "terms.prohibited.title",
    paragraphs: ["terms.prohibited.body", "terms.prohibited.action"],
  },
  {
    title: "terms.availability.title",
    paragraphs: ["terms.availability.body"],
  },
  { title: "terms.liability.title", paragraphs: ["terms.liability.body"] },
  { title: "terms.changes.title", paragraphs: ["terms.changes.body"] },
  {
    title: "terms.contact.title",
    paragraphs: ["terms.contact.body", "legal.launchNote"],
  },
];

export default function TermsPage() {
  return (
    <LegalPage
      label="terms.label"
      title="terms.title"
      intro="terms.intro"
      pageTitle="terms.pageTitle"
      sections={sections}
    />
  );
}
