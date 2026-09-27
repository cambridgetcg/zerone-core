import catalog from "./module-guide.json";

const repository = "https://github.com/cambridgetcg/zerone-core";

/** Explanatory data shared by the existing HTML and JSON guides, not app configuration. */
export function buildModuleGuide(commit: string) {
  const source = (path: string, kind = "blob") => ({ label: path, path, url: `${repository}/${kind}/${commit}/${path}` });
  return {
    title: catalog.title,
    intro: catalog.intro,
    registration: { label: "Application registration and wiring", url: `${repository}/blob/${commit}/app/app.go` },
    development: {
      ...catalog.development,
      tracks: catalog.development.tracks.map(({ link, ...track }) => ({
        ...track,
        link: { label: link.label, url: link.url ?? source(link.path!).url },
      })),
    },
    groups: catalog.groups,
    modules: catalog.modules.map(({ sourcePath, ...module }) => ({ ...module, tags: ["registered-source", module.family, module.group], source: source(sourcePath) })),
    workflow: catalog.workflow.map(({ link, ...step }) => ({
      ...step,
      link: { label: link.label, url: link.url ?? source(link.path!).url },
    })),
    boundaries: catalog.boundaries,
    sharedUtilities: catalog.sharedUtilities.map(({ sourcePath, sourceKind, ...utility }) => ({ ...utility, source: source(sourcePath, sourceKind) })),
  };
}
