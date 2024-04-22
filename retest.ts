

import * as core from '@actions/core'
import * as github from '@actions/github'
import axios from 'axios'
import { OctokitResponse } from "@octokit/types";
import { GithubReactionType, CheckRunsType, CreateReactionType, Retest, RetestResult, PR, Env } from './types'

function cachedProperty(_: unknown, key: string, descriptor: PropertyDescriptor): PropertyDescriptor {
  const originalGetter = descriptor.get
  if (!originalGetter) {
    throw new Error('The decorated property must have a getter.')
  }


  const cachedValueKey = Symbol(`__cached_${key}`)

  descriptor.get = function (this: any): any {

    if (!this[cachedValueKey]) {
      this[cachedValueKey] = originalGetter.call(this)
    }

    return this[cachedValueKey]
  }

  return descriptor
}

class RetestCommand {

  public checks: CheckRunsType['data']['check_runs']
  public env: Env
  public name = ''
  public pr: PR

  constructor(env: Env, pr: PR, checks: CheckRunsType['data']['check_runs']) {
    this.env = env
    this.checks = checks
    this.pr = pr
  }

  retest = async (): Promise<RetestResult> => {
    if (!this.env) {
      core.warning(`Failed parsing env`)
      return {errors: 0, retested: 0}
    }
    const retestables = await this.getRetestables()
    if (Object.keys(retestables).length === 0) {
      return {errors: 0, retested: 0}
    }
    return await this.retestRuns(retestables)
  }

  retestExternal = async (check: Retest): Promise<number> => {
    if (check.method == 'patch') {
      try {
        await axios.patch(check.url, {}, check.config)
        /* eslint-disable  prettier/prettier */
      } catch (error: any) {
        if (!axios.isAxiosError(error) || !error.response) {
          core.error('No response received')
          return 1
        }
        core.error(`External API call failed: ${check.url}`)
        core.error(error.response.data.message)
        return 1
      }
    }
    return 0
  }

  retestOctokit = async (pr: PR, check: Retest): Promise<number> => {

    const method = check.method || 'POST'
    const rerunURL = `${method} ${check.url}`
    if (rerunURL.endsWith('rerun-failed-jobs')) {
      console.log(`Retesting failed job (pr #${pr.number}): ${check.name}`)
    } else {
      console.log(`Restarting check (pr #${pr.number}): ${check.name}`)
    }
    const rerunResponse = await this.env.octokit.request(rerunURL, check.config || {})
    if ([200, 201].includes(rerunResponse.status)) {
      if (rerunURL.endsWith('rerun-failed-jobs')) {
        process.stdout.write(`::notice::Retry success: (${check.name})\n`)
      } else {
        process.stdout.write(`::notice::Check restarted: (${check.name})\n  ${rerunResponse.data.html_url}\n`)
      }
      return 0
    } else {
      if (rerunURL.endsWith('rerun-failed-jobs')) {
        core.error(`Retry failed: (${check.name}) ... ${rerunResponse.status}`)
      } else {
        core.error(`Failed restarting check: ${rerunResponse.status}`)
      }
      return 1
    }
  }

  retestRuns = async (retestables: Array<Retest>): Promise<RetestResult> => {

    let errors = 0
    for (const check of retestables) {
      if (!check.octokit) {
        errors += await this.retestExternal(check)
      } else {
        errors += await this.retestOctokit(this.pr, check)
      }
    }
    return {retested: Object.keys(retestables).length, errors: errors}
  }

  getRetestables = async (): Promise<Array<Retest>> => {
    return []
  }
}

class GithubRetestCommand extends RetestCommand {

  name = 'Github'

  constructor(env: Env, pr: PR, checks: CheckRunsType['data']['check_runs']) {
    super(env, pr, checks)
  }

  getRetestables = async (): Promise<Array<Retest>> => {
    const failedChecks: any[] = []
    const checks = this.checks.filter((checkRun) => {
      return checkRun.app?.slug
    })
    checks.forEach(async (check: any) => {
      if (check.conclusion !== 'failure' && check.conclusion !== 'cancelled') {
        return
      }
      if (this.env.debug) {
        console.log(
          `Check ${check.conclusion}: ${check.name}\n\n  ${check.html_url}\n\n`
          + `  https://github.com/${this.env.owner}/${this.env.repo}/actions/runs/${check.external_id}\n`)
      }
      // run failed jobs.
      failedChecks.push({
        name: check.name || 'unknown',
        url: `/repos/${this.env.owner}/${this.env.repo}/actions/runs/${check.external_id}/rerun-failed-jobs`,
        octokit: true,
      })
      // TODO: Update the old check to mention restart
      // Create a new check from the old
      const toDelete = ['pull_requests', 'app', 'check_suite', 'conclusion', 'node_id', 'started_at', 'completed_at', 'id']
      Object.keys(check).forEach((key) => {
        if (key.startsWith("url") || key.endsWith("url") || toDelete.includes(key)) {
          delete check[key]
        }
      })
      Object.keys(check.output).forEach((key) => {
        if (key.startsWith("annotation")) {
          delete check.output[key]
        }
      })

      check.output.title = check.output.title.replace('failure', 'restarted')
      const lines = check.output.text.split('\n')
      const line0 = lines[0].replace('Check run finished (failure :x:)', 'Check run restarted')
      check.output.text = `${line0}\n${lines.slice(1).join('\n')}`
      check.output.summary = 'Check is running again'
      check.status = 'in_progress'

      failedChecks.push({
        name: check.name || 'unknown',
        url: `/repos/${this.env.owner}/${this.env.repo}/check-runs`,
        octokit: true,
        config: {data: check},
      })
    })
    return failedChecks
  }
}

class RetestCommands {

  @cachedProperty
  get env(): Env {
    const token = core.getInput('token') || process.env['GITHUB_TOKEN']
    if (!token || token === '') throw new TypeError('`token` must be set')

    const pr = core.getInput('pr-url')
    const comment = parseInt(core.getInput('comment-id'))
    console.log(token)
    const octokit = github.getOctokit(token)
    // Create the octokit client
    const nwo = process.env['GITHUB_REPOSITORY'] || '/'
    const [owner, repo] = nwo.split('/')
    const debug = Boolean(process.env.CI_DEBUG && process.env.CI_DEBUG != 'false')

    return {
      debug,
      token,
      octokit,
      nwo,
      owner,
      repo,
      comment,
      pr,
    }
  }

  checks = async (pr: PR): Promise<CheckRunsType['data']['check_runs']> => {
    const response: CheckRunsType = await this.env.octokit.rest.checks.listForRef({
      owner: this.env.owner,
      per_page: 100,
      repo: this.env.repo,
      ref: pr.commit,
      filter: 'latest',
    })
    const checks = response.data.check_runs
    if (!checks) return []
    if (this.env.debug) {
      checks.forEach((checkRun) => {
        console.log(`Found check (${checkRun.id}/${checkRun.app?.slug}/${checkRun.conclusion || 'incomplete'}): ${checkRun.name}`);
      });
    }
    return checks
  }

  getPR = async (): Promise<PR | void> => {
    if (!this.env.pr || !this.env.pr) {
      return
    }
    const response: OctokitResponse<any> = await this.env.octokit.request(this.env.pr)
    const data = response.data
    if (!data) return
    if (this.env.debug) {
      console.log(`Running /retest command for PR #${data.number}`)
      console.log(`PR branch: ${data.head.ref}`)
      console.log(`Latest PR commit: ${data.head.sha}`)
    }
    return {
      number: data.number,
      branch: data.head.ref,
      commit: data.head.sha,
    }
  }

  retesters = async (): Promise<Array<RetestCommand>> => {
    if (!this.env) {
      return []
    }
    const pr = await this.getPR()
    if (!pr) {
      return []
    }
    const checks = await this.checks(pr)
    const retesters: Array<RetestCommand> = [new GithubRetestCommand(this.env, pr, checks)]
    return retesters
  }

  addReaction = async (reaction: GithubReactionType = 'rocket'): Promise<void> => {
    const addReactionResponse: CreateReactionType['response'] = await this.env.octokit.rest.reactions.createForIssueComment({
      owner: this.env.owner,
      repo: this.env.repo,
      comment_id: this.env.comment,
      content: reaction,
    })
    if ([200, 201].includes(addReactionResponse.status)) {
      console.log(`Reacted to comment ${reaction}`)
    } else {
      core.error(`Failed reacting to comment ${reaction}`)
    }
  }

  retest = async (): Promise<void> => {
    const result: RetestResult = {errors: 0, retested: 0}
    for (const retester of await this.retesters()) {
      const retested = await retester.retest()
      result.errors += retested.errors
      result.retested += retested.retested
    }
    if (result.errors !== 0) {
      await this.addReaction('-1')
    }
    if (result.retested === 0) {
      await this.addReaction('confused')
    } else {
      await this.addReaction()
    }
  }
}

const run = async (): Promise<void> => {
  try {
    const retesters = new RetestCommands()
    await retesters.retest()
  } catch (error) {
    if (error instanceof Error) {
      console.error(error.message)
    }
    core.setFailed(`retest-action failure: ${error}`)
  }
}

// Don't auto-execute in the test environment
if (process.env['NODE_ENV'] !== 'test') {

  run()
}

export default run
