// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT
import {pttclient} from '../models';
import {context} from '../models';

export function Close():Promise<void>;

export function Connect():Promise<void>;

export function FetchPostMessages(arg1:string,arg2:string):Promise<any>;

export function GotoBoard(arg1:string):Promise<any>;

export function Lock():Promise<void>;

export function Login(arg1:string,arg2:string):Promise<void>;

export function SendMessage(arg1:pttclient.MessageType,arg2:string):Promise<void>;

export function StartUp(arg1:context.Context):Promise<void>;

export function Unlock():Promise<void>;

export function Wait():Promise<void>;