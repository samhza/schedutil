import json
import time
import aiohttp
import asyncio
import ijson
import urllib
import statistics
import sys
from dataclasses import dataclass
from datetime import datetime
from enum import Enum
from collections import namedtuple
from typing import List
import itertools

courses = {}
indexes = {}
open_sections = {}
SOC_URL = "https://classes.rutgers.edu/soc"
SOC_API_URL = SOC_URL + "/api"

CAC = '1'
BUSCH = '2'
LIVI = '3'
ONLINE = 'O'

MONDAY = 'M'
TUESDAY = 'T'
WEDNESDAY = 'W'
THURSDAY = 'H'
FRIDAY = 'F'

DAYS = [MONDAY, TUESDAY, WEDNESDAY, THURSDAY, FRIDAY]

@dataclass
class Meet:
    day: str
    location: str
    start: int
    end: int
    course_title: str

    def compatible_with(a, b: 'Meet'):
        if a.day != b.day:
            return True
        same_campus = a.location == b.location or \
                         a.location == ONLINE or b.location == ONLINE or \
                        (a.location == BUSCH and b.location == LIVI) or \
                        (b.location == BUSCH and a.location == LIVI)
        required_gap = 30 if same_campus else 60
        if max(a.start, b.start) - min(a.end, b.end) >= required_gap:
            return True
        return False


@dataclass
class Section:
    index: str
    meets: List[Meet]

    def compatible_with(a, b: 'Section'):
        meet_products = itertools.product(a.meets, b.meets)
        return all(c.compatible_with(d) for c, d in meet_products)

registered = {}

async def main():
    async with aiohttp.ClientSession() as s:
        td = await get_term_date(s)
        await update_courses(s, td)
        await get_open(s, td)
    my_courses = ["01:198:112", "01:198:205", "01:750:124", "01:355:101"]
    sections = {}
    indexes = []
    for css in my_courses:
        secs = []
        if not type(css) is list:
            css = [css]
        for cs in css:
            for sec in courses[cs]["sections"]:
                if sec["index"].startswith("H") and cs == "01:640:152":
                    continue
                if sec["index"] not in open_sections and sec["index"] not in registered:
                    continue
                meets = []
                dont_add = False
                for m in sec["meetingTimes"]:
                    if m["meetingDay"] == "F":
                        dont_add = True
                    if m["campusLocation"] not in [BUSCH, LIVI, ]:#CAC]:
                        dont_add = True
                    meet = Meet(
                        day=m["meetingDay"],
                        start=parse_military(m["startTimeMilitary"]),
                        end=parse_military(m["endTimeMilitary"]),
                        location=m["campusLocation"],
                        course_title=courses[cs]["title"])
                    if meet.day == THURSDAY and meet.end > 17*60+10:
                        dont_add = True
                    meets.append(meet)
                if dont_add:
                    continue
                sections[int(sec["index"])] = meets
                secs.append(Section(index=sec["index"], meets=meets))
        indexes.append(secs)
    n = 0
    sorted(indexes, key=lambda i:len(i))
    for sched in valid_schedules(*indexes):
        meets = [m for sched in sched for m in sched.meets]
        meets = []
        for m in sched:
            meets.extend(m.meets)
        if daily_transfers_exceed(meets,3):
            continue
        if len(maxdaily(meets)) > 4:
            continue
        #score = sum(n_ for day in DAYS for m in meets if m.day == day)
        #score = day_length(m)
        score = day_length(meets)
        #score += 100000000 * sum(n_transfers(m for m in meets if m.day == day) for day in DAYS)
        print(score, end=" ")
        print(*[f"{x.day}{x.location}{x.start},{x.end}={x.course_title}" for x in meets], sep=':', end='§')
        print(*[x.index for x in sched], sep=',')

def n_transfers(meets: List[Meet]):
    meets = sorted(meets, key=lambda x: x.start)
    loc = BUSCH
    n = 0
    for meet in meets:
        if meet.location != ONLINE and loc != meet.location:
            loc = meet.location
            n += 1
    if loc != BUSCH:
        n += 1
    return n

def daily_transfers_exceed(meets: List[Meet], max: int) -> bool:
    daily_meets = [[m for m in meets if m.day == day] for day in DAYS]
    return any(n_transfers(m) > max for m in daily_meets)

def valid_schedules(*indexes):
    print(len(indexes), file=sys.stderr)
    invalid = -1
    invalididx = 0
    invalid2 = 0
    invalididx2 = 0
    prevvalid = []
    for sched in itertools.product(*indexes):
        if invalid != -1 and sched[invalididx].index == invalid and sched[invalididx2].index == invalid2:
            continue
        invalid = 0
        for section1, section2 in itertools.combinations(enumerate(sched), 2):
            i1, s1 = section1
            i2, s2 = section2
            if prevvalid and prevvalid[i1] == s1.index and prevvalid[i2] == s2.index:
                continue
            if not s1.compatible_with(s2):
                invalid = s2.index
                invalididx = i2
                invalid2 = s1.index
                invalididx2 = i1
                break
            if invalid:
                break
        if invalid:
            continue
        prevvalid = [x.index for x in sched]
        yield sched

def maxdaily(meets: List[Meet]):
    daily_classes = ([m for m in meets if m.day == d] for d in DAYS)
    return max(daily_classes, key=lambda m: len(m))

def bus_rides(meets, home=''):
    n = 0
    day_meets = [sorted([m for m in meets if m.day == day], key=lambda m:m.start) for day in DAYS]
    for meets in day_meets:
        loc = '2'
        for meet in meets:
            if meet.location != loc or meet.location != ONLINE:
                loc = meet.location
                n += 1
    if not home:
        return n-2
    return n

def day_lengths(meets):
    starts = {}
    ends = {}
    for meet in meets:
        if meet.day not in ends or meet.end > ends[meet.day]:
            ends[meet.day] = meet.end
        if meet.day not in starts or meet.start < starts[meet.day]:
            starts[meet.day] = meet.start
    return [ends[day] - start for day, start in starts.items()]


def day_length(meets):
    return sum(day_lengths(meets))

def avg_endtime(meets):
    ends = {}
    for meet in meets:
        if meet.day not in ends or meet.end > ends[meet.day]:
            ends[meet.day] = meet.end
    return sum(ends.values())/len(ends)

def parse_military(t: str) -> int:
    if len(t) == 0:
        return -1
    hours = int(t[:2])
    minutes = int(t[2:])
    return hours*60 + minutes

async def get_open(s, td):
    global open_sections
    params = {
        'year': td['year'],
        'term': td['term'],
        'campus': td['campus']
    }
    print(f"{SOC_API_URL}/openSections.json", file=sys.stderr)
    r = await s.get(f"{SOC_API_URL}/openSections.json", params=params)
    open_sections = set(await r.json())

async def get_term_date(s):
    async with s.get(SOC_URL) as r:
        PREFIX = b'<div id="initJsonData" style="display:none;">'
        soc_text = None
        async for line in r.content:
            if not line.startswith(PREFIX):
                continue
            soc_text = line.rstrip()[len(PREFIX):-len("</div>")]
            break
        term_date = json.loads(soc_text.decode(r.charset))["currentTermDate"]
        term_date = {
            "campus": "NB",
            "term": "1",
            "year": '2024',
        }
        return term_date

async def update_courses(s, td):
    params = {
        'year': td['year'],
        'term': td['term'],
        'campus': td['campus']
    }
    with open("2024.json") as r:
        for course in ijson.items(r, 'item'):
            courses[course["courseString"]] = course
    return

def fmt_section(index):
    course = courses[indexes[index]]
    title = course["title"]
    section_n = course["sections"][index]
    return f"{title} section {section_n} ({index})"

if __name__ == '__main__':
    loop = asyncio.get_event_loop()
    loop.run_until_complete(main())
