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

