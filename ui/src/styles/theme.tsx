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

const DEFAULT_SPACING = 8;

export const MOLT_THEME = createTheme({
    spacing: DEFAULT_SPACING,
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
        MuiOutlinedInput: {
            styleOverrides: {
                root: {
                    "& .MuiOutlinedInput-notchedOutline": {
                        border: `1px solid ${neutral[500]}`,
                    },
                    "&.Mui-focused": {
                        "& .MuiOutlinedInput-notchedOutline": {
                            border: `2px solid ${info[3]}`,
                        },
                    },
                    "&.Mui-error": {
                        backgroundColor: danger[1],
                        color: danger[4],
                        "& .MuiOutlinedInput-notchedOutline": {
                            border: `1px solid ${danger[4]}`,
                        },
                    },
                    "&.Mui-disabled": {
                        backgroundColor: neutral[100],
                        color: neutral[900],
                        "& .MuiOutlinedInput-notchedOutline": {
                            border: `1px solid ${neutral[400]}`,
                        },
                    },
                },
            }
        },
        MuiFormHelperText: {
            styleOverrides: {
                root: {
                    '&.MuiFormHelperText-root': {
                        color: danger[4],
                        padding: 0,
                        margin: 0
                    }
                },
            },
        },
        MuiFormControlLabel: {
            styleOverrides: {
                label: {
                    fontSize: '0.9rem',
                    color: neutral[600],
                }
            }
        },
        MuiSwitch: {
            styleOverrides: {
                root: {
                    width: DEFAULT_SPACING * 5,
                    height: DEFAULT_SPACING * 3,
                    padding: 0,
                    display: 'flex',
                    '&:active': {
                        '& .MuiSwitch-thumb': {
                            width: 15,
                        },
                        '& .MuiSwitch-switchBase.Mui-checked': {
                            transform: 'translateX(20px)',
                        },
                    },
                },
                track: {
                    // Controls default (unchecked) color for the track
                    borderRadius: DEFAULT_SPACING * 3 / 2,
                    opacity: 1,
                    backgroundColor: neutral[400],
                    boxSizing: 'border-box',
                },
                switchBase: {
                    padding: 2,
                    '&.Mui-checked': {
                        transform: `translateX(${DEFAULT_SPACING * 2}px)`,
                        color: '#fff',
                        '& + .MuiSwitch-track': {
                            opacity: 1,
                            backgroundColor: info[2]
                        },
                    },
                },
                // This needs to be after the track so track styles take precedence first.
                thumb: {
                    // TODO: figure out how to change this when checked.
                    border: `1px solid ${neutral[200]}`,
                    boxShadow: '0 2px 4px 0 rgb(0 35 11 / 20%)',
                    width: 20,
                    height: 20,
                    borderRadius: 10,
                }
            }
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