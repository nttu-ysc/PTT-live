export namespace consts {
	
	export class Post {
	    url: string;
	    aid: string;
	    sn: string;
	    title: string;
	    author: string;
	    push_count: string;
	    date: string;
	
	    static createFrom(source: any = {}) {
	        return new Post(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.aid = source["aid"];
	        this.sn = source["sn"];
	        this.title = source["title"];
	        this.author = source["author"];
	        this.push_count = source["push_count"];
	        this.date = source["date"];
	    }
	}

}

export namespace pttclient {
	
	export class Message {
	    // Go type: time
	    time: any;
	    content: string;
	    author: string;
	    hash: string;
	
	    static createFrom(source: any = {}) {
	        return new Message(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = this.convertValues(source["time"], null);
	        this.content = source["content"];
	        this.author = source["author"];
	        this.hash = source["hash"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace pttcrawler {
	
	export class HotBoard {
	    name: string;
	    user_count: string;
	    category: string;
	
	    static createFrom(source: any = {}) {
	        return new HotBoard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.user_count = source["user_count"];
	        this.category = source["category"];
	    }
	}

}

