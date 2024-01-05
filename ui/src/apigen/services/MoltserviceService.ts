/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { CreateFetchTaskRequestBody } from '../models/CreateFetchTaskRequestBody';
import type { FetchRun } from '../models/FetchRun';
import type { FetchRunDetailed } from '../models/FetchRunDetailed';

import type { CancelablePromise } from '../core/CancelablePromise';
import { OpenAPI } from '../core/OpenAPI';
import { request as __request } from '../core/request';

export class MoltserviceService {

    /**
     * get_fetch_tasks moltservice
     * @returns FetchRun OK response.
     * @throws ApiError
     */
    public static moltserviceGetFetchTasks(): CancelablePromise<Array<FetchRun>> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/fetch',
        });
    }

    /**
     * create_fetch_task moltservice
     * @param requestBody
     * @returns number OK response.
     * @throws ApiError
     */
    public static moltserviceCreateFetchTask(
        requestBody: CreateFetchTaskRequestBody,
    ): CancelablePromise<number> {
        return __request(OpenAPI, {
            method: 'POST',
            url: '/api/v1/fetch',
            body: requestBody,
            mediaType: 'application/json',
        });
    }

    /**
     * get_specific_fetch_task moltservice
     * @param id id for the fetch task
     * @returns FetchRunDetailed OK response.
     * @throws ApiError
     */
    public static moltserviceGetSpecificFetchTask(
        id: number,
    ): CancelablePromise<FetchRunDetailed> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/fetch/{id}',
            path: {
                'id': id,
            },
        });
    }

    /**
     * create_verify_task_from_fetch moltservice
     * @param id id for the fetch task
     * @returns number OK response.
     * @throws ApiError
     */
    public static moltserviceCreateVerifyTaskFromFetch(
        id: number,
    ): CancelablePromise<number> {
        return __request(OpenAPI, {
            method: 'POST',
            url: '/api/v1/fetch/{id}/verify',
            path: {
                'id': id,
            },
        });
    }

    /**
     * Download ./assets/docs.html
     * @returns any File downloaded
     * @throws ApiError
     */
    public static moltserviceDocsHtml(): CancelablePromise<any> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/docs.html',
        });
    }

    /**
     * Download ./gen/http/openapi.json
     * @returns any File downloaded
     * @throws ApiError
     */
    public static moltserviceOpenapiJson(): CancelablePromise<any> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/openapi.json',
        });
    }

}
