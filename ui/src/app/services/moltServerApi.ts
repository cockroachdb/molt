import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import { getFetchTasks } from '../../api';
import { FetchListRun, getStatusFromString } from '../../pages/FetchList';
import { formatSecondsToHHMMSS } from '../../utils/dates';

export const moltServerApi = createApi({
    reducerPath: 'moltServerApi',
    // TODO: make this dependent on the process environment variable later.
    baseQuery: fetchBaseQuery({ baseUrl: 'http://localhost:8000/' }),
    endpoints: (build) => ({
        getFetchRuns: build.query<FetchListRun[], void>({
            queryFn: async () => {
                const fetchTasks = await getFetchTasks()

                const fetchListRuns: FetchListRun[] = fetchTasks.map(item => {
                    const startedAtTs = new Date(item.started_at * 1000);
                    const finishedAtTs = new Date(item.finished_at * 1000);

                    return {
                        key: `${item.id.toString()}-${crypto.randomUUID()}`,
                        id: item.id.toString(),
                        name: item.name,
                        status: getStatusFromString(item.status),
                        duration: formatSecondsToHHMMSS(item.finished_at - item.started_at),
                        startedAt: startedAtTs.toISOString(),
                        finishedAt: finishedAtTs.toISOString(),
                        errors: 0,
                    }
                })

                return { data: fetchListRuns }
            },
        }),
    }),
});

export const { useGetFetchRunsQuery } = moltServerApi;
