import { createContext, Dispatch, SetStateAction, useContext, useEffect, useMemo, useState } from 'react';

import '../index.css';
import { Box } from "@mui/material";
import { Outlet, useOutletContext } from "react-router-dom";
import LeftDrawer from "./LeftDrawer";
import { ActiveFilters } from './FilterBar';
import { AuthInfo, CommandResultSummary, ShortName, ValidateResultSummary } from "../models";
import { Api, checkStaticBuild, RealApi, StaticApi, User } from "../api";
import { buildProjectSummaries, ProjectSummary } from "../project-summaries";
import Login from "./Login";
import { Loading, useLoadingHelper } from "./Loading";
import { ErrorMessageCard } from './ErrorMessage';
import { useImmer } from "use-immer";

export interface AppOutletContext {
    filters?: ActiveFilters
    setFilters: Dispatch<SetStateAction<ActiveFilters | undefined>>
}
export function useAppOutletContext(): AppOutletContext {
    return useOutletContext<AppOutletContext>()
}

export interface AppContextProps {
    api: Api
    user: User
    authInfo: AuthInfo
    isStatic: boolean
    filters?: ActiveFilters
    commandResultSummaries: Map<string, CommandResultSummary>
    projects: ProjectSummary[]
    validateResultSummaries: Map<string, ValidateResultSummary>
    shortNames: ShortName[]
}
export const AppContext = createContext<AppContextProps | undefined>(undefined);
export function useAppContext() {
    return useContext(AppContext)!
}

export interface KluctlDeploymentWithClusterId {
    deployment: any
    clusterId: string
}

const LoggedInApp = (props: { api: Api, user: User, authInfo: AuthInfo, isStatic: boolean, onLogout: () => void }) => {
    const [filters, setFilters] = useState<ActiveFilters>()

    const [commandResultSummaries, setCommandResultSummaries] = useImmer(new Map<string, CommandResultSummary>())
    const [validateResultSummaries, setValidateResultSummaries] = useImmer(new Map<string, ValidateResultSummary>())
    const [kluctlDeployments, setKluctlDeployments] = useImmer(new Map<string, KluctlDeploymentWithClusterId>())

    useEffect(() => {
        const updateCommandResultSummary = (rs: CommandResultSummary) => {
            console.log("update_command_result_summary", rs.id, rs.commandInfo.startTime)
            setCommandResultSummaries(draft => {
                draft.set(rs.id, rs)
            })
        }

        const deleteCommandResultSummary = (id: string) => {
            console.log("delete_command_result_summary", id)
            setCommandResultSummaries(draft => {
                draft.delete(id)
            })
        }

        const updateValidateResultSummary = (vr: ValidateResultSummary) => {
            console.log("update_validate_result_summary", vr.id)
            setValidateResultSummaries(draft => {
                draft.set(vr.id, vr)
            })
        }

        const deleteValidateResultSummary = (id: string) => {
            console.log("delete_validate_result_summary", id)
            setValidateResultSummaries(draft => {
                draft.delete(id)
            })
        }

        const updateKluctlDeployment = (kd: any, clusterId: string) => {
            console.log("update_kluctl_deployment", kd.metadata.uid, kd.metadata.name)
            setKluctlDeployments(draft => {
                draft.set(kd.metadata.uid, {
                    deployment: kd,
                    clusterId: clusterId,
                })
            })
        }

        const deleteKluctlDeployment = (id: string) => {
            console.log("delete_kluctl_deployment", id)
            setKluctlDeployments(draft => {
                draft.delete(id)
            })
        }

        console.log("starting listenResults")
        let cancel: Promise<() => void>
        cancel = props.api.listenEvents(undefined, undefined, msg => {
            switch (msg.type) {
                case "update_command_result_summary":
                    updateCommandResultSummary(msg.summary)
                    break
                case "delete_command_result_summary":
                    deleteCommandResultSummary(msg.id)
                    break
                case "update_validate_result_summary":
                    updateValidateResultSummary(msg.summary)
                    break
                case "delete_validate_result_summary":
                    deleteValidateResultSummary(msg.id)
                    break
                case "update_kluctl_deployment":
                    updateKluctlDeployment(msg.deployment, msg.clusterId)
                    break
                case "delete_kluctl_deployment":
                    deleteKluctlDeployment(msg.id)
                    break
            }
        })
        return () => {
            console.log("cancel listenResults")
            cancel.then(c => c())
        }
    }, [props.api, setCommandResultSummaries, setValidateResultSummaries, setKluctlDeployments])

    const projects = useMemo(() => {
        return buildProjectSummaries(commandResultSummaries, validateResultSummaries, kluctlDeployments, filters)
    }, [commandResultSummaries, validateResultSummaries, kluctlDeployments, filters])

    const [loading, loadingError, shortNames] = useLoadingHelper<ShortName[]>(true,
        () => props.api.getShortNames(),
        [props.api]
    );

    const appCtx: AppContextProps = useMemo(() => {
        return {
            api: props.api,
            user: props.user,
            authInfo: props.authInfo,
            isStatic: props.isStatic,
            filters: filters,
            commandResultSummaries: commandResultSummaries,
            projects,
            validateResultSummaries: validateResultSummaries,
            shortNames: shortNames || []
        }
    }, [
        props.api, props.user, props.authInfo, props.isStatic,
        filters, commandResultSummaries, projects, validateResultSummaries, shortNames,
    ])

    if (loading) {
        return <Loading />;
    }

    if (loadingError) {
        return <ErrorMessageCard>
            {loadingError.message}
        </ErrorMessageCard>;
    }

    const outletContext: AppOutletContext = {
        filters: filters,
        setFilters: setFilters,
    }

    return (
        <AppContext.Provider value={appCtx}>
            <Box width={"100%"} height={"100%"}>
                <LeftDrawer
                    content={<Outlet context={outletContext} />}
                    context={outletContext}
                    logout={props.onLogout}
                />
            </Box>
        </AppContext.Provider>
    );
};

const App = () => {
    const [api, setApi] = useState<Api>()
    const [authInfo, setAuthInfo] = useState<AuthInfo>()
    const [user, setUser] = useState<User>()
    const [isStatic, setIsStatic] = useState(false)

    const onLogout = () => {
        console.log("handle onLogout")
        setUser(undefined)
        const params = new URLSearchParams()
        params.set("returnUrl", `${window.location.protocol}//${window.location.host}`)
        window.location.href='/auth/logout?' + params.toString()
    }
    const onUnauthorized = () => {
        console.log("handle onUnauthorized")
        setUser(undefined)
    }

    useEffect(() => {
        if (api) {
            return
        }

        const doInit = async () => {
            console.log("checking if this is a static webui build")
            const isStatic = await checkStaticBuild()
            console.log("isStatic=" + isStatic)
            let api: Api
            if (isStatic) {
                api = new StaticApi()
            } else {
                api = new RealApi(onUnauthorized)
            }
            const authInfo = await api.getAuthInfo()
            setApi(api)
            setAuthInfo(authInfo)
            setIsStatic(isStatic)
        }
        doInit()
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    useEffect(() => {
        if (user || !api) {
            return
        }
        const doGetUser = async () => {
            if (!api) {
                return
            }
            try {
                const user = await api.getUser()
                console.log("user", user)
                setUser(user)
            } catch (error) {
                console.log("error", error)
            }
        }
        doGetUser()
    }, [user, api])

    if (!api || !authInfo) {
        return <Loading />
    }

    if (!user) {
        return <Login authInfo={authInfo} />
    }

    return <LoggedInApp onLogout={onLogout} api={api} user={user} authInfo={authInfo} isStatic={isStatic}/>
}

export default App;
