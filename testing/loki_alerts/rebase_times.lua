tag_first_timestamps = {}
tag_first_seen_wall_clock_times = {}

function cb_rebase_times(tag, timestamp, record)
    for test_tag, first_ts in pairs(tag_first_timestamps) do
        if tag == test_tag then
            new_ts = (timestamp - first_ts) + tag_first_seen_wall_clock_times[tag]
            return 1, new_ts, record
        end
    end

    tag_first_timestamps[tag] = timestamp
    tag_first_seen_wall_clock_times[tag] = os.time() - (60 * 30) -- shift it back an hour, just in case
    return 0, timestamp, record
end
