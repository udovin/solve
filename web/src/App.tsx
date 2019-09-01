import React from "react";
import {BrowserRouter, Switch, Route} from "react-router-dom";
import "./App.scss";
import Header from "./layout/Header";
import Footer from "./layout/Footer";
import LoginPage from "./pages/LoginPage";
import RegisterPage from "./pages/RegisterPage";
import IndexPage from "./pages/IndexPage";
import ContestsPage from "./pages/ContestsPage";
import LanguagePage from "./pages/LanguagePage";
import {AuthProvider} from "./AuthContext";
import ProblemPage from "./pages/ProblemPage";

const App: React.FC = () => {
	return (
		<div id="layout">
			<AuthProvider>
				<BrowserRouter>
					<Header/>
					<Switch>
						<Route exact path="/" component={IndexPage}/>
						<Route exact path="/contests" component={ContestsPage}/>
						<Route exact path="/login" component={LoginPage}/>
						<Route exact path="/register" component={RegisterPage}/>
						<Route exact path="/language" component={LanguagePage}/>
						<Route exact path="/problems/:ProblemID" component={ProblemPage}/>
					</Switch>
					<Footer/>
				</BrowserRouter>
			</AuthProvider>
		</div>
	);
};

export default App;
