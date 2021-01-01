import React from "react";
import Button from "./index";
import "../../index.scss";

const config = {title: "Button"};

export default config;

export const common = () => <>
	<p><Button>Default</Button></p>
	<p><Button disabled>Disabled</Button></p>
	<p><Button color="primary">Primary</Button></p>
	<p><Button color="success">Success</Button></p>
	<p><Button color="danger">Danger</Button></p>
</>;
