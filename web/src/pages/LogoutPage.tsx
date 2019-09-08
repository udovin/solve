import React, {useContext, useEffect, useState} from "react";
import {Redirect} from "react-router";
import {AuthContext} from "../AuthContext";

const LogoutPage = () => {
	const {session, setSession} = useContext(AuthContext);
	const [success, setSuccess] = useState<boolean>();
	useEffect(() => {
		if (session) {
			fetch("/api/v0/sessions/" + session.ID, {
				method: "DELETE",
				headers: {
					"Content-Type": "application/json; charset=UTF-8",
				},
			})
				.then(() => {
					setSuccess(true);
					setSession();
				});
		}
	}, [session, setSession]);
	if (!session || success) {
		return <Redirect to={"/"}/>;
	}
	return <>Loading...</>;
};

export default LogoutPage;
