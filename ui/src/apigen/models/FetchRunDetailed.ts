/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

import type { FetchStatsDetailed } from './FetchStatsDetailed';
import type { Log } from './Log';

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
    status: string;
};

