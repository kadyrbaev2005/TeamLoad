// Step lists for the AI loading card (wording from the design).
import { fmtDate } from "./format";

export const designTeamSteps = (deadline: string, size: number) => [
  "Reading the project description",
  "Estimating work until " + fmtDate(deadline),
  `Choosing ${size} roles`,
  "Writing responsibilities and skills",
];

export const redesignSteps = (hasNote: boolean, seats: number) => [
  hasNote ? "Reading your note" : "Re-reading the description",
  `Balancing effort across ${seats} seats`,
  "Writing new rationales",
];

export const matchingSteps = (people: number) => [
  `Reading ${people} profiles`,
  "Scoring every person for every role",
  "Finding the best fit for the whole team",
  "Writing explanations",
];

export const raciSteps = ["Reading project and roles", "Ordering activities to the deadline", "Checking one A per activity"];
