/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

import type { VerifyMismatch } from './VerifyMismatch';
import type { VerifyStatsDetailed } from './VerifyStatsDetailed';

export type VerifyRunDetailed = {
    /**
     * ID of the associated fetch run
     */
    fetch_id: number;
    /**
     * finished at time
     */
    finished_at: number;
    /**
     * ID of the run
     */
    id: number;
    /**
     * verify mismatches (i.e. data mismatches, missing rows)
     */
    mismatches: Array<VerifyMismatch>;
    /**
     * name of the run
     */
    name: string;
    /**
     * started at time
     */
    started_at: number;
    stats: VerifyStatsDetailed;
    /**
     * status of the run
     */
    status: VerifyRunDetailed.status;
};

export namespace VerifyRunDetailed {

    /**
     * status of the run
     */
    export enum status {
        IN_PROGRESS = 'IN_PROGRESS',
        SUCCESS = 'SUCCESS',
        FAILURE = 'FAILURE',
    }


}

