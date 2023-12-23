import { createTheme } from '@mui/material/styles';
import { danger, info, primary, success, warning } from './colors';
import { fontSizes } from './fonts';

export const MOLT_THEME = createTheme({
    spacing: 8,
    // TODO: figure out the font size/weights for each typography component.
    typography: {
        fontFamily: [
            'Source Sans Pro',
            'sans-serif'
        ].join(','),
        fontSize: fontSizes["md"]
    },
    palette: {
        primary: {
            main: primary[3],
            light: primary[1],
            dark: primary[5],
        },
        secondary: {
            main: primary[2],
            light: primary[1],
            dark: primary[3],
        },
        success: {
            main: success[3],
            light: success[1],
            dark: success[5],
        },
        info: {
            main: info[3],
            light: info[1],
            dark: info[5],
        },
        error: {
            main: danger[3],
            light: danger[1],
            dark: danger[5],
        },
        warning: {
            main: warning[3],
            light: warning[1],
            dark: warning[5],
        }
    },
});