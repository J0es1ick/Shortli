import LegalPage, { type LegalSection } from "./legalPage";

const sections: LegalSection[] = [
  {
    title: "privacy.data.title",
    paragraphs: [
      "privacy.data.account",
      "privacy.data.links",
      "privacy.data.analytics",
    ],
  },
  {
    title: "privacy.purpose.title",
    paragraphs: ["privacy.purpose.service", "privacy.purpose.security"],
  },
  { title: "privacy.cookies.title", paragraphs: ["privacy.cookies.body"] },
  {
    title: "privacy.storage.title",
    paragraphs: ["privacy.storage.body", "privacy.storage.delete"],
  },
  { title: "privacy.sharing.title", paragraphs: ["privacy.sharing.body"] },
  { title: "privacy.rights.title", paragraphs: ["privacy.rights.body"] },
  {
    title: "privacy.contact.title",
    paragraphs: ["privacy.contact.body", "legal.launchNote"],
  },
];

export default function PrivacyPage() {
  return (
    <LegalPage
      label="privacy.label"
      title="privacy.title"
      intro="privacy.intro"
      pageTitle="privacy.pageTitle"
      sections={sections}
    />
  );
}
