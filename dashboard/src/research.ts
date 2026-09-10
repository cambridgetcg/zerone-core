import "./styles.css";
import { initialiseBranchFlow } from "./branch-flow";
import { initialiseConstructiveTree } from "./constructive-tree";
import { initialiseResearchCommons } from "./research-commons";
import { initialiseLifeSciencesTree } from "./life-sciences-tree";
import { initialiseQuantumSeason } from "./quantum-season";
import { initialiseMathFrontier } from "./math-frontier";
import { initialiseFoldToFire } from "./fold-to-fire";
import { initialiseLifeGarden } from "./life-garden";
import { initialiseRelationalTopology } from "./relational-topology";
import { initialiseCorrespondenceGeometry } from "./correspondence-geometry";
import { initialiseExplicitInvariantDiscipline } from "./explicit-invariant-discipline";

const byId = (id: string): HTMLElement => {
  const element = document.getElementById(id);
  if (!element) throw new Error(`Missing #${id}`);
  return element;
};
const branchFlowRoot = byId("branch-flow-root");
const constructiveTreeRoot = byId("constructive-tree-root");
const researchCommonsRoot = byId("research-commons-root");
const lifeSciencesTreeRoot = byId("life-sciences-tree-root");
const quantumSeasonRoot = byId("quantum-season-root");
const mathFrontierRoot = byId("math-frontier-root");
const foldToFireRoot = byId("fold-to-fire-root");
const lifeGardenRoot = byId("life-garden-root");
const relationalTopologyRoot = byId("relational-topology-root");
const correspondenceGeometryRoot = byId("correspondence-geometry-root");
const explicitInvariantDisciplineRoot = byId("explicit-invariant-discipline-root");

// The library loads only same-origin pinned static artifacts. It has no wallet,
// account pilot, live-chain client, transaction handler or network refresh timer.
document.querySelectorAll<HTMLElement>(".reveal").forEach((element) => element.classList.add("is-visible"));
const initialViews = Promise.allSettled([
  initialiseBranchFlow(branchFlowRoot),
  initialiseConstructiveTree(constructiveTreeRoot),
  initialiseResearchCommons(researchCommonsRoot),
  initialiseLifeSciencesTree(lifeSciencesTreeRoot),
  initialiseQuantumSeason(quantumSeasonRoot),
  initialiseMathFrontier(mathFrontierRoot),
  initialiseFoldToFire(foldToFireRoot),
  initialiseLifeGarden(lifeGardenRoot),
  initialiseRelationalTopology(relationalTopologyRoot),
  initialiseCorrespondenceGeometry(correspondenceGeometryRoot),
  initialiseExplicitInvariantDiscipline(explicitInvariantDisciplineRoot),
]);
const alignResearchHash = (): void => {
  const id = window.location.hash.slice(1);
  if (!id) return;
  document.getElementById(id)?.scrollIntoView({ block: "start", behavior: "instant" });
};
void initialViews.then(alignResearchHash);
window.addEventListener("hashchange", () => { void initialViews.then(alignResearchHash); });
