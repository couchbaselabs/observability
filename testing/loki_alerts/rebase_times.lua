tag_first_timestamps = {}
tag_first_seen_wall_clock_times = {}

 ns_in_1s = 1000000000

function add_time_tables(t1, t2)
    local new_s = t1["sec"] + t2["sec"]
    local new_ns = t1["nsec"] + t2["nsec"]
    if new_ns > ns_in_1s then
        new_s = new_s + math.floor(new_ns / ns_in_1s)
        new_ns = new_ns % ns_in_1s
    end
    return {sec=new_s, nsec=new_ns}
end

function sub_time_tables(t1, t2)
    local new_s = t1["sec"] - t2["sec"]
    local new_ns = t1["nsec"] - t2["nsec"]
    if new_ns < 0 then
        new_s = new_s + math.floor(new_ns / ns_in_1s)
        new_ns = new_ns % ns_in_1s
    end
    return {sec=new_s, nsec=new_ns}
end

function cb_rebase_times(tag, timestamp, record)
    local found = false
    for test_tag, first_ts in pairs(tag_first_timestamps) do
        if tag == test_tag then
            found = true
            break
        end
    end

    if not found then
        tag_first_timestamps[tag] = timestamp
        tag_first_seen_wall_clock_times[tag] = math.floor(os.time() - (60 * 30)) -- shift it back an hour, just in case
    end

    local new_ts = add_time_tables(sub_time_tables(timestamp, tag_first_timestamps[tag]), {sec=tag_first_seen_wall_clock_times[tag], nsec=0})
    -- The "timestamp" record key (string) will now be nonsense, so remove it - downstream should be using the Loki timestamp anyway
    record["timestamp"] = nil
    return 1, new_ts, record
end
