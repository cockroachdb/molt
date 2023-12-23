import { createTheme } from '@mui/material/styles';
import { danger, info, neutral, primary, success, warning } from './colors';
import { fontSizes } from './fonts';

// Button override.
declare module '@mui/material/Button' {
    interface ButtonPropsVariantOverrides {
        secondary: true;
        tertiary: true;
    }
}

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
        },
        action: {
            disabledBackground: primary[2],
            disabled: neutral[0]
        }
    },
    components: {
        MuiButtonBase: {
            defaultProps: {
                disableRipple: true,
            },
        },
        MuiButton: {
            variants: [
                {
                    props: { variant: 'secondary' },
                    style: {
                        textTransform: 'none',
                        color: neutral[900],
                        backgroundColor: neutral[0],
                        border: `1px solid ${neutral[500]}`,
                        "&:hover": {
                            backgroundColor: neutral[100],
                            border: `1px solid ${neutral[700]}`,
                        },
                        "&:active": {
                            backgroundColor: neutral[100],
                            border: `1px solid ${neutral[700]}`,
                        },
                        "&:focus": {
                            backgroundColor: neutral[100],
                            border: `1px solid ${neutral[700]}`,
                        },
                        "&:disabled": {
                            backgroundColor: neutral[0],
                            border: `1px solid ${neutral[300]}`,
                            color: neutral[600],
                        }
                    },
                },
                {
                    props: { variant: 'tertiary' },
                    style: {
                        textTransform: 'none',
                        backgroundColor: neutral[0],
                        border: `1px solid ${neutral[0]}`,
                        color: info[3],
                        "&:hover": {
                            backgroundColor: neutral[0],
                            color: info[4],
                        },
                        "&:active": {
                            backgroundColor: neutral[0],
                            color: info[3],
                            border: `1px solid ${info[1]}`,
                        },
                        "&:focus": {
                            backgroundColor: neutral[0],
                            color: info[3],
                            border: `1px solid ${info[3]}`,
                        },
                        "&:disabled": {
                            color: info[2],
                        }
                    },
                },
            ],
        },
    }
});