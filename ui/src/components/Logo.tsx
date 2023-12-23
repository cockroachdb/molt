import Box from '@mui/material/Box';
import cockroachLogoDefault from "../assets/images/logos/cockroach-logo-default.svg"
import cockroachLogoFullColorDarkBackground from "../assets/images/logos/cockroach-logo-full-color-dark-background.svg";

export type LogoColor =
    | "default"
    | "reduced-color-dark-background"
    | "reduced-color-light-background"
    | "full-color-dark-background"
    | "white"
    | "black";

export type LogoSize =
    | "default"
    | "xs"
    | "small"
    | "smedium"
    | "medium"
    | "large"
    | "xlg";

type LogoType = "cockroach" | "cockroach-color-dark-bg";

interface LogoProps {
    className?: string;
    // lockup and superuser correspond to the full symbol and name
    type?: LogoType;
    // should be set to default/large for lockup type
    size?: LogoSize;
    // default corresponds to the full-color-light-background
    color?: LogoColor;
    path?: string;
}

const getLogoFromType = (type: LogoType) => {
    if (type === "cockroach") {
        return cockroachLogoDefault;
    } else if (type === "cockroach-color-dark-bg") {
        return cockroachLogoFullColorDarkBackground;
    }
}

const getHeightFromSize = (size: LogoSize) => {
    if (["small", "medium", "smedium", "default"].includes(size)) {
        return "26px";
    }

    if (size === "large") {
        return "36px";
    }

    if (size === "xlg") {
        return "44px"
    }

    return "26px"
}

const Logo = (props: LogoProps): JSX.Element => {
    const {
        type = "cockroach",
        size = "default",
    } = props;

    return (
        <Box
            sx={{
                height: getHeightFromSize(size)
            }}
            component="img"
            src={getLogoFromType(type)}
        />
    );
};

export default Logo;