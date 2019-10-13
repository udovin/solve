import React, {FC, useEffect, useState} from "react";
import {CurrentSession} from "./api";

type Auth = {
	session?: CurrentSession;
	setSession(session?: CurrentSession): void;
};

const AuthContext = React.createContext<Auth>({
	setSession(): void {}
});

const AuthProvider: FC = props => {
	const [session, setSession] = useState<CurrentSession>();
	useEffect(() => {
		fetch("/api/v0/sessions/current")
			.then(result => result.json())
			.then(result => setSession(result))
			.catch(error => setSession(undefined))
	}, []);
	return <AuthContext.Provider value={{session, setSession}}>
		{props.children}
	</AuthContext.Provider>;
};

export {AuthContext, AuthProvider};
