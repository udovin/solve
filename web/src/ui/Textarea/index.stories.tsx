import React from "react";
import Textarea from "./index";

export default {title: "Textarea"};

export const common = () => <>
	<p><Textarea defaultValue="Default"/></p>
	<p><Textarea defaultValue="Disabled" disabled/></p>
</>;
