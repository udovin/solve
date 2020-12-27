import React, {useContext, useEffect, useState} from "react";
import {Redirect} from "react-router";
import {AuthContext} from "../AuthContext";

const LogoutPage = () => {
	const {status, setStatus} = useContext(AuthContext);
	const [success, setSuccess] = useState<boolean>();
	useEffect(() => {
		if (status) {
			fetch("/api/v0/logout", {
				method: "POST",
				headers: {
					"Content-Type": "application/json; charset=UTF-8",
				},
			})
				.then(() => {
					setSuccess(true);
					setStatus();
				});
		}
	}, [status, setStatus]);
	if (!(status && status.user) || success) {
		return <Redirect to={"/"}/>;
	}
	return <>Loading...</>;
};

export default LogoutPage;
