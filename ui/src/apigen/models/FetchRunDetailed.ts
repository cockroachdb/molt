/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

import type { FetchStatsDetailed } from './FetchStatsDetailed';
import type { Log } from './Log';
import type { VerifyRun } from './VerifyRun';

export type FetchRunDetailed = {
    /**
     * finished at time
     */
    finished_at: number;
    /**
     * ID of the run
     */
    id: number;
    /**
     * logs for fetch run
     */
    logs: Array<Log>;
    /**
     * name of the fetch run
     */
    name: string;
    /**
     * started at time
     */
    started_at: number;
    stats?: FetchStatsDetailed;
    /**
     * status of the fetch run
     */
    status: FetchRunDetailed.status;
    /**
     * verify runs linked to fetch runs
     */
    verify_runs: Array<VerifyRun>;
};

export namespace FetchRunDetailed {

    /**
     * status of the fetch run
     */
    export enum status {
        IN_PROGRESS = 'IN_PROGRESS',
        SUCCESS = 'SUCCESS',
        FAILURE = 'FAILURE',
    }


}

