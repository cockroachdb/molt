import { configureStore } from '@reduxjs/toolkit';
import { setupListeners } from '@reduxjs/toolkit/query';
import { moltServerApi } from './services/moltServerApi';

export const store = configureStore({
    reducer: {
        [moltServerApi.reducerPath]: moltServerApi.reducer,
    },
    middleware: (getDefaultMiddleware) =>
        getDefaultMiddleware().concat(moltServerApi.middleware),
});

setupListeners(store.dispatch);