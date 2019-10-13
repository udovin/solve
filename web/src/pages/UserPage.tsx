import React, {useEffect, useState} from "react";
import {RouteComponentProps} from "react-router";
import Page from "../components/Page";
import {User} from "../api";
import Block from "../components/Block";
import "./ContestPage.scss"
import Sidebar from "../components/Sidebar";
import Field from "../components/Field";

type UserPageParams = {
	UserID: string;
}

const UserPage = ({match}: RouteComponentProps<UserPageParams>) => {
	const {UserID} = match.params;
	const [user, setUser] = useState<User>();
	useEffect(() => {
		fetch("/api/v0/users/" + UserID)
			.then(result => result.json())
			.then(result => setUser(result));
	}, [UserID]);
	if (!user) {
		return <>Loading...</>;
	}
	const {Login, FirstName, LastName, MiddleName} = user;
	return <Page title={Login} sidebar={<Sidebar/>}>
		<Block title={Login} id="block-user">
			{FirstName && <Field title="First name:"><span>{FirstName}</span></Field>}
			{LastName && <Field title="Last name:"><span>{LastName}</span></Field>}
			{MiddleName && <Field title="Middle name:"><span>{MiddleName}</span></Field>}
		</Block>
	</Page>;
};

export default UserPage;
