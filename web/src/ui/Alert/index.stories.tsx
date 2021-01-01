import React from "react";
import Alert, {AlertKind} from "./index";

const config = {title: "Alert"};

export default config;

export const common = () => <>
	<Alert>Default</Alert>
	<Alert kind={AlertKind.DANGER}>Danger</Alert>
	<Alert kind={AlertKind.INFO}>Info</Alert>
	<Alert kind={AlertKind.WARNING}>Warning</Alert>
	<Alert kind={AlertKind.SUCCESS}>Success</Alert>
</>;
