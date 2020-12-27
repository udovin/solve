import React, {useEffect, useState} from "react";
import {RouteComponentProps} from "react-router";
import Page from "../components/Page";
import {User} from "../api";
import Block from "../ui/Block";
import "./ContestPage.scss"
import Sidebar from "../components/Sidebar";
import Field from "../ui/Field";

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
	const {login, email, first_name, last_name, middle_name} = user;
	return <Page title={login} sidebar={<Sidebar/>}>
		<Block title={login} id="block-user">
			{first_name && <Field title="E-mail:"><span>{email}</span></Field>}
			{first_name && <Field title="First name:"><span>{first_name}</span></Field>}
			{last_name && <Field title="Last name:"><span>{last_name}</span></Field>}
			{middle_name && <Field title="Middle name:"><span>{middle_name}</span></Field>}
		</Block>
	</Page>;
};

export default UserPage;
