import { GitHub } from "@actions/github/lib/utils";
import { RestEndpointMethodTypes } from "@octokit/rest";
import { Endpoints } from "@octokit/types";

type GithubReactionType =
  | "rocket"
  | "+1"
  | "-1"
  | "laugh"
  | "confused"
  | "heart"
  | "hooray"
  | "eyes";
  
type CheckRunsType =
  Endpoints["GET /repos/{owner}/{repo}/commits/{ref}/check-runs"]["response"];
type CreateReactionType =
  RestEndpointMethodTypes["reactions"]["createForIssueComment"];

type Retest = {
  name: string;
  octokit: boolean;
  url: string;
  method?: string;
  config?: any;
};

type RetestResult = {
  retested: number;
  errors: number;
};

type PR = {
  number: number;
  branch: string;
  commit: string;
};

type OctokitType = InstanceType<typeof GitHub>;

type Env = {
  octokit: OctokitType;
  token: string;
  comment: number;
  debug: boolean;
  pr: string;
  nwo: string;
  owner: string;
  repo: string;
  appOwnerSlug: string;
  azpOrg: string | undefined;
  azpOwnerSlug: string;
  azpToken: string | undefined;
};

export {
  GithubReactionType,
  CheckRunsType,
  CreateReactionType,
  Retest,
  RetestResult,
  PR,
  OctokitType,
  Env,
};
